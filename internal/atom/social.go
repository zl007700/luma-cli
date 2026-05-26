package atom

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"html"
	"io"
	"mime"
	"net/http"
	"net/url"
	"os"
	"path/filepath"
	"regexp"
	"sort"
	"strings"
	"time"
)

const (
	defaultDouyinUserAgent = "Mozilla/5.0 (Macintosh; Intel Mac OS X 10_15_7) AppleWebKit/537.36 (KHTML, like Gecko) Chrome/131.0.0.0 Safari/537.36"
	mobileDouyinUserAgent  = "Mozilla/5.0 (iPhone; CPU iPhone OS 17_2 like Mac OS X) AppleWebKit/605.1.15 (KHTML, like Gecko) EdgiOS/121.0.2277.107 Version/17.0 Mobile/15E148 Safari/604.1"
	defaultDouyinAccept    = "text/html,application/xhtml+xml,application/xml;q=0.9,image/avif,image/webp,image/apng,*/*;q=0.8,application/signed-exchange;v=b3;q=0.7"
	defaultDouyinPageURL   = "https://www.douyin.com/user/MS4wLjABAAAAN_s_hups7LD0N4qnrM3o2gI0vuG3pozNaEolz2_py3cHTTrpVr1Z4dukFD9SOlwY?from_tab_name=main&modal_id=%s"

	envDouyinCookie = "LUMA_DOUYIN_COOKIE"
	envDouyinPaths  = "LUMA_DOUYIN_PATHS"
)

var (
	douyinURLPattern        = regexp.MustCompile(`https?://[^\s"'<>]+`)
	douyinModalIDPattern    = regexp.MustCompile(`modal_id[=:](\d+)`)
	douyinVideoIDPattern    = regexp.MustCompile(`/video/(\d+)`)
	douyinNoteIDPattern     = regexp.MustCompile(`/note/(\d+)`)
	douyinPlainIDPattern    = regexp.MustCompile(`^\d{16,}$`)
	renderDataPattern       = regexp.MustCompile(`(?s)<script\s+id=["']RENDER_DATA["']\s+type=["']application/json["']>(.*?)</script>`)
	jingxuanJSONPattern     = regexp.MustCompile(`"([^"]*?(?:playAddr|searchProps|app)[^"]*?)"`)
	routerDataPattern       = regexp.MustCompile(`(?s)window\._ROUTER_DATA\s*=\s*(.*?)</script>`)
	windowsInvalidNameChars = regexp.MustCompile(`[<>:"/\\|?*\x00-\x1f]`)
)

type SocialDownloadResult struct {
	VideoPath string `json:"video_path"`
	Title     string `json:"title"`
	VideoID   string `json:"video_id"`
	Path      string `json:"path"`
	VideoURL  string `json:"video_url,omitempty"`
}

// DownloadSocialVideo downloads a Douyin video from a share link.
func DownloadSocialVideo(shareLink, outputPath, cardKey string) (*SocialDownloadResult, error) {
	ctx, cancel := context.WithTimeout(context.Background(), 4*time.Minute)
	defer cancel()

	client := &http.Client{Timeout: 35 * time.Second}
	resolver := douyinResolver{
		client: client,
		headers: douyinHeaders{
			Accept:    defaultDouyinAccept,
			Cookie:    loadDouyinCookie(cardKey),
			UserAgent: defaultDouyinUserAgent,
		},
	}

	modalID, err := resolver.extractModalID(ctx, shareLink)
	if err != nil {
		return nil, fmt.Errorf("extract modal_id: %w", err)
	}

	videoURL, title, pathName, err := resolver.resolveVideo(ctx, modalID)
	if err != nil {
		return nil, fmt.Errorf("get video url: %w", err)
	}
	if videoURL == "" {
		return nil, errors.New("no playable video URL (private, deleted, or blocked by Douyin)")
	}

	if outputPath == "" {
		outputPath = sanitizeDouyinFilename(title, modalID) + ".mp4"
	}
	absOut, err := filepath.Abs(outputPath)
	if err != nil {
		return nil, fmt.Errorf("resolve output path: %w", err)
	}
	if err := os.MkdirAll(filepath.Dir(absOut), 0755); err != nil {
		return nil, fmt.Errorf("create output dir: %w", err)
	}
	if err := resolver.downloadFile(ctx, videoURL, absOut); err != nil {
		return nil, fmt.Errorf("download video: %w", err)
	}

	return &SocialDownloadResult{
		VideoPath: absOut,
		Title:     title,
		VideoID:   modalID,
		Path:      pathName,
		VideoURL:  videoURL,
	}, nil
}

type douyinHeaders struct {
	Accept    string
	Cookie    string
	UserAgent string
}

type douyinResolver struct {
	client  *http.Client
	headers douyinHeaders
}

func (r douyinResolver) resolveVideo(ctx context.Context, modalID string) (videoURL, title, pathName string, err error) {
	type resolvePath struct {
		name string
		fn   func(context.Context, string) (string, string, error)
	}
	allPaths := map[string]resolvePath{
		"jingxuan": {name: "douyin_jingxuan", fn: r.getVideoURLByJingxuan},
		"ies":      {name: "iesdouyin_router_data", fn: r.getVideoURLByIESRouterData},
		"cookie":   {name: "user_modal_cookie", fn: r.getVideoURLByUserModal},
	}
	pathOrder := []string{"jingxuan", "ies", "cookie"}
	if forced := strings.TrimSpace(os.Getenv(envDouyinPaths)); forced != "" {
		pathOrder = strings.Split(forced, ",")
	}

	var errs []string
	for _, key := range pathOrder {
		key = strings.ToLower(strings.TrimSpace(key))
		item, ok := allPaths[key]
		if !ok {
			errs = append(errs, key+": unknown path")
			continue
		}
		if key == "cookie" && strings.TrimSpace(r.headers.Cookie) == "" {
			errs = append(errs, item.name+": cookie not available")
			continue
		}
		videoURL, title, err := item.fn(ctx, modalID)
		if err == nil && videoURL != "" {
			return videoURL, title, item.name, nil
		}
		if err != nil {
			errs = append(errs, item.name+": "+err.Error())
		}
	}
	return "", "", "", fmt.Errorf("all local paths failed: %s", strings.Join(errs, " | "))
}

func (r douyinResolver) extractModalID(ctx context.Context, input string) (string, error) {
	input = strings.TrimSpace(input)
	if modalID := extractDouyinVideoID(input); modalID != "" {
		return modalID, nil
	}

	shareURL := firstDouyinURL(input)
	if shareURL == "" {
		return "", fmt.Errorf("no Douyin URL found")
	}

	req, err := http.NewRequestWithContext(ctx, http.MethodGet, shareURL, nil)
	if err != nil {
		return "", err
	}
	req.Header.Set("User-Agent", r.headers.UserAgent)

	resp, err := r.client.Do(req)
	if err != nil {
		return "", fmt.Errorf("request share link: %w", err)
	}
	defer resp.Body.Close()

	finalURL := resp.Request.URL.String()
	if modalID := extractDouyinVideoID(finalURL); modalID != "" {
		return modalID, nil
	}
	return "", fmt.Errorf("no video id found after redirect: %s", finalURL)
}

func firstDouyinURL(input string) string {
	for _, raw := range douyinURLPattern.FindAllString(input, -1) {
		cleaned := strings.TrimRight(raw, " \t\r\n.,;!?，。！？）)]】》\"'")
		if strings.Contains(cleaned, "douyin.com") {
			return cleaned
		}
	}
	return ""
}

func extractDouyinVideoID(value string) string {
	value = strings.TrimSpace(value)
	for _, pattern := range []*regexp.Regexp{douyinModalIDPattern, douyinVideoIDPattern, douyinNoteIDPattern} {
		if m := pattern.FindStringSubmatch(value); len(m) == 2 {
			return m[1]
		}
	}
	if douyinPlainIDPattern.MatchString(value) {
		return value
	}
	return ""
}

func (r douyinResolver) getVideoURLByJingxuan(ctx context.Context, modalID string) (string, string, error) {
	pageURL := "https://www.douyin.com/jingxuan?modal_id=" + url.QueryEscape(modalID)
	body, err := r.fetchPage(ctx, pageURL, r.headers.UserAgent, false)
	if err != nil {
		return "", "", err
	}
	detail, err := parseJingxuanVideoDetail(body)
	if err != nil {
		return "", "", err
	}
	videoURL, err := bestDouyinPlayURL(detail)
	if err != nil {
		return "", "", err
	}
	return videoURL, detailTitle(detail), nil
}

func (r douyinResolver) getVideoURLByIESRouterData(ctx context.Context, modalID string) (string, string, error) {
	pageURL := "https://www.iesdouyin.com/share/video/" + url.PathEscape(modalID) + "/"
	body, err := r.fetchPage(ctx, pageURL, mobileDouyinUserAgent, false)
	if err != nil {
		return "", "", err
	}
	detail, err := parseIESRouterVideoDetail(body)
	if err != nil {
		return "", "", err
	}
	videoURL, err := bestIESPlayURL(detail)
	if err != nil {
		return "", "", err
	}
	return videoURL, detailTitle(detail), nil
}

func (r douyinResolver) getVideoURLByUserModal(ctx context.Context, modalID string) (string, string, error) {
	pageURL := fmt.Sprintf(defaultDouyinPageURL, url.QueryEscape(modalID))
	body, err := r.fetchPage(ctx, pageURL, r.headers.UserAgent, true)
	if err != nil {
		return "", "", err
	}
	detail, err := parseRenderDataVideoDetail(body)
	if err != nil {
		return "", "", err
	}
	videoURL, err := bestDouyinPlayURL(detail)
	if err != nil {
		return "", "", err
	}
	return videoURL, detailTitle(detail), nil
}

func (r douyinResolver) fetchPage(ctx context.Context, pageURL, userAgent string, includeCookie bool) (string, error) {
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, pageURL, nil)
	if err != nil {
		return "", err
	}
	req.Header.Set("Accept", r.headers.Accept)
	req.Header.Set("Accept-Language", "zh-CN,zh;q=0.9,en;q=0.8")
	req.Header.Set("Referer", "https://www.douyin.com/")
	req.Header.Set("User-Agent", userAgent)
	if includeCookie && strings.TrimSpace(r.headers.Cookie) != "" {
		req.Header.Set("Cookie", r.headers.Cookie)
	}

	resp, err := r.client.Do(req)
	if err != nil {
		return "", fmt.Errorf("request Douyin page: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode >= 400 {
		return "", fmt.Errorf("Douyin page HTTP %d", resp.StatusCode)
	}
	data, err := io.ReadAll(io.LimitReader(resp.Body, 16<<20))
	if err != nil {
		return "", fmt.Errorf("read Douyin page: %w", err)
	}
	return string(data), nil
}

func (r douyinResolver) downloadFile(ctx context.Context, srcURL, dstPath string) error {
	srcURL = strings.TrimSpace(strings.NewReplacer("\r", "", "\n", "", "\t", "").Replace(srcURL))
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, srcURL, nil)
	if err != nil {
		return err
	}
	req.Header.Set("Referer", "https://www.douyin.com/")
	req.Header.Set("User-Agent", r.headers.UserAgent)

	resp, err := r.client.Do(req)
	if err != nil {
		return fmt.Errorf("GET %s: %w", srcURL, err)
	}
	defer resp.Body.Close()

	if resp.StatusCode >= 400 {
		return fmt.Errorf("HTTP %d", resp.StatusCode)
	}
	if ct := resp.Header.Get("Content-Type"); ct != "" {
		mediaType, _, _ := mime.ParseMediaType(ct)
		if strings.HasPrefix(mediaType, "text/html") {
			return fmt.Errorf("download returned HTML instead of video")
		}
	}

	f, err := os.Create(dstPath)
	if err != nil {
		return fmt.Errorf("create %s: %w", dstPath, err)
	}
	defer f.Close()

	n, err := io.Copy(f, resp.Body)
	if err != nil {
		return fmt.Errorf("write %s: %w", dstPath, err)
	}
	if n == 0 {
		return fmt.Errorf("downloaded empty file")
	}
	return nil
}

func parseJingxuanVideoDetail(pageHTML string) (map[string]any, error) {
	matches := jingxuanJSONPattern.FindAllStringSubmatch(pageHTML, -1)
	for _, match := range matches {
		if len(match) != 2 {
			continue
		}
		if !strings.Contains(match[1], "playAddr") || !strings.Contains(match[1], "app") {
			continue
		}
		decoded, err := url.QueryUnescape(html.UnescapeString(match[1]))
		if err != nil {
			continue
		}
		var root map[string]any
		if err := json.Unmarshal([]byte(decoded), &root); err != nil {
			continue
		}
		if detail := appVideoDetail(root); len(detail) > 0 {
			return detail, nil
		}
	}
	return nil, errors.New("jingxuan page did not include playable video JSON")
}

func parseRenderDataVideoDetail(pageHTML string) (map[string]any, error) {
	matches := renderDataPattern.FindStringSubmatch(pageHTML)
	if len(matches) != 2 {
		return nil, errors.New("page did not include RENDER_DATA")
	}
	decoded, err := url.QueryUnescape(html.UnescapeString(matches[1]))
	if err != nil {
		return nil, fmt.Errorf("decode RENDER_DATA: %w", err)
	}
	var root map[string]any
	if err := json.Unmarshal([]byte(decoded), &root); err != nil {
		return nil, fmt.Errorf("parse RENDER_DATA JSON: %w", err)
	}
	detail := appVideoDetail(root)
	if len(detail) == 0 {
		return nil, errors.New("RENDER_DATA did not include app.videoDetail")
	}
	return detail, nil
}

func parseIESRouterVideoDetail(pageHTML string) (map[string]any, error) {
	matches := routerDataPattern.FindStringSubmatch(pageHTML)
	if len(matches) != 2 {
		return nil, errors.New("iesdouyin page did not include window._ROUTER_DATA")
	}
	raw := strings.TrimSpace(strings.TrimSuffix(matches[1], ";"))
	var root map[string]any
	if err := json.Unmarshal([]byte(raw), &root); err != nil {
		return nil, fmt.Errorf("parse window._ROUTER_DATA JSON: %w", err)
	}

	loader, _ := root["loaderData"].(map[string]any)
	if loader == nil {
		loader = root
	}
	for _, key := range []string{"video_(id)/page", "note_(id)/page"} {
		page, _ := loader[key].(map[string]any)
		info, _ := page["videoInfoRes"].(map[string]any)
		items, _ := info["item_list"].([]any)
		if len(items) == 0 {
			continue
		}
		detail, _ := items[0].(map[string]any)
		if len(detail) > 0 {
			return detail, nil
		}
	}
	return nil, errors.New("window._ROUTER_DATA did not include videoInfoRes.item_list")
}

func appVideoDetail(root map[string]any) map[string]any {
	app, _ := root["app"].(map[string]any)
	detail, _ := app["videoDetail"].(map[string]any)
	return detail
}

func detailTitle(videoDetail map[string]any) string {
	title, _ := videoDetail["desc"].(string)
	if strings.TrimSpace(title) == "" {
		return "douyin_video"
	}
	return title
}

func bestDouyinPlayURL(videoDetail map[string]any) (string, error) {
	video, _ := videoDetail["video"].(map[string]any)
	if url := firstPlayAddrURL(video["playAddr"]); url != "" {
		return url, nil
	}

	bitrates, _ := video["bitRateList"].([]any)
	if len(bitrates) == 0 {
		return "", errors.New("video did not include playAddr or bitRateList")
	}
	sort.SliceStable(bitrates, func(i, j int) bool {
		left, _ := bitrates[i].(map[string]any)
		right, _ := bitrates[j].(map[string]any)
		return numberValue(left["bitRate"]) > numberValue(right["bitRate"])
	})
	for _, item := range bitrates {
		rate, _ := item.(map[string]any)
		if url := firstPlayAddrURL(rate["playAddr"]); url != "" {
			return url, nil
		}
	}
	return "", errors.New("video did not include a playable URL")
}

func bestIESPlayURL(videoDetail map[string]any) (string, error) {
	video, _ := videoDetail["video"].(map[string]any)
	playAddr, _ := video["play_addr"].(map[string]any)
	if url := firstURLListItem(playAddr["url_list"]); url != "" {
		return strings.Replace(url, "playwm", "play", 1), nil
	}
	downloadAddr, _ := video["download_addr"].(map[string]any)
	if url := firstURLListItem(downloadAddr["url_list"]); url != "" {
		return strings.Replace(url, "playwm", "play", 1), nil
	}
	return "", errors.New("iesdouyin data did not include play_addr.url_list")
}

