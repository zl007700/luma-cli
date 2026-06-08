package atom

import (
	"net/url"
	"testing"
)

func TestExtractDouyinVideoID(t *testing.T) {
	got := extractDouyinVideoID("https://www.douyin.com/video/7523380123456789012")
	if got != "7523380123456789012" {
		t.Fatalf("got %q", got)
	}
}

func TestParseDouyinVideoDetailAndBestURL(t *testing.T) {
	renderData := `{"app":{"videoDetail":{"desc":"demo title","video":{"bitRateList":[{"bitRate":100,"playAddr":[{"src":"//example.com/low.mp4"}]},{"bitRate":200,"playAddr":[{"src":"https://example.com/high.mp4"}]}]}}}}`
	page := `<html><script id="RENDER_DATA" type="application/json">` + url.QueryEscape(renderData) + `</script></html>`

	detail, err := parseRenderDataVideoDetail(page)
	if err != nil {
		t.Fatalf("parse detail: %v", err)
	}
	if detail["desc"] != "demo title" {
		t.Fatalf("unexpected title: %#v", detail["desc"])
	}

	playURL, err := bestDouyinPlayURL(detail)
	if err != nil {
		t.Fatalf("best url: %v", err)
	}
	if playURL != "https://example.com/high.mp4" {
		t.Fatalf("unexpected url: %s", playURL)
	}
}

func TestParseJingxuanVideoDetail(t *testing.T) {
	renderData := `{"app":{"videoDetail":{"desc":"demo title","video":{"playAddr":[{"src":"https://example.com/direct.mp4"}]}}},"searchProps":{}}`
	page := `<html>"` + url.QueryEscape(renderData) + `"</html>`

	detail, err := parseJingxuanVideoDetail(page)
	if err != nil {
		t.Fatalf("parse jingxuan detail: %v", err)
	}
	playURL, err := bestDouyinPlayURL(detail)
	if err != nil {
		t.Fatalf("best url: %v", err)
	}
	if playURL != "https://example.com/direct.mp4" {
		t.Fatalf("unexpected url: %s", playURL)
	}
}

func TestParseIESRouterVideoDetail(t *testing.T) {
	page := `<script>window._ROUTER_DATA = {"loaderData":{"video_(id)/page":{"videoInfoRes":{"item_list":[{"desc":"demo title","video":{"play_addr":{"url_list":["https://example.com/playwm.mp4"]}}}]}}}};</script>`

	detail, err := parseIESRouterVideoDetail(page)
	if err != nil {
		t.Fatalf("parse ies detail: %v", err)
	}
	playURL, err := bestIESPlayURL(detail)
	if err != nil {
		t.Fatalf("best ies url: %v", err)
	}
	if playURL != "https://example.com/play.mp4" {
		t.Fatalf("unexpected url: %s", playURL)
	}
}

func TestSanitizeDouyinFilename(t *testing.T) {
	got := sanitizeDouyinFilename(`  a/b:c* demo  `, "123")
	if got != "a_b_c_ demo" {
		t.Fatalf("unexpected sanitized name: %q", got)
	}
}

func TestIsCompleteDownload(t *testing.T) {
	if !isCompleteDownload(1024, 1024) {
		t.Fatal("expected exact content length to be complete")
	}
	if !isCompleteDownload(2048, 1024) {
		t.Fatal("expected larger written size to be complete")
	}
	if isCompleteDownload(512, 1024) {
		t.Fatal("partial download should not be complete")
	}
	if isCompleteDownload(1024, -1) {
		t.Fatal("unknown content length should not be treated as complete")
	}
}
