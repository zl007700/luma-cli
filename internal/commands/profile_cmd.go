package commands

import (
	"encoding/json"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"regexp"
	"sort"
	"strings"
	"time"

	"github.com/luma-cli/lumer-cli/cloud"
	"github.com/luma-cli/lumer-cli/internal/atom"
	"github.com/luma-cli/lumer-cli/internal/cmdutil"
	appconfig "github.com/luma-cli/lumer-cli/internal/config"
	"github.com/luma-cli/lumer-cli/internal/output"
)

var profileIDPattern = regexp.MustCompile(`^[A-Za-z0-9][A-Za-z0-9._-]*$`)

type lumaProfile struct {
	ID         string         `json:"id"`
	Identity   string         `json:"identity"`
	Audience   []string       `json:"audience"`
	Stance     []string       `json:"stance"`
	Avoid      []string       `json:"avoid"`
	AssetGroup string         `json:"asset_group,omitempty"`
	Assets     []profileAsset `json:"assets,omitempty"`
	CreatedAt  time.Time      `json:"created_at"`
	UpdatedAt  time.Time      `json:"updated_at"`
	ExtraPath  string         `json:"extra_path,omitempty"`
}

type profileAsset struct {
	ID        string    `json:"id"`
	Kind      string    `json:"kind,omitempty"`
	Name      string    `json:"name,omitempty"`
	Group     string    `json:"group,omitempty"`
	ObjectKey string    `json:"object_key,omitempty"`
	LocalPath string    `json:"local_path,omitempty"`
	CreatedAt time.Time `json:"created_at"`
}

type profileView struct {
	lumaProfile
	Extra string `json:"extra,omitempty"`
	Path  string `json:"path,omitempty"`
}

type profileState struct {
	CurrentProfile string `json:"current_profile,omitempty"`
}

func cmdProfile(args []string) error {
	if len(args) < 1 {
		printProfileUsage()
		return nil
	}
	switch args[0] {
	case "create":
		return cmdProfileCreate(args[1:])
	case "update":
		return cmdProfileUpdate(args[1:])
	case "list", "ls":
		return cmdProfileList(args[1:])
	case "get", "show":
		return cmdProfileGet(args[1:])
	case "use":
		return cmdProfileUse(args[1:])
	case "current":
		return cmdProfileCurrent(args[1:])
	case "asset", "assets":
		return cmdProfileAsset(args[1:])
	default:
		printProfileUsage()
		return nil
	}
}

func cmdProfileCreate(raw []string) error {
	args := cmdutil.Parse(raw)
	id := strings.TrimSpace(args.Pos(0))
	if id == "" {
		id = strings.TrimSpace(args.String("id", ""))
	}
	if err := validateProfileID(id); err != nil {
		return output.ErrValidation("%v", err)
	}
	profile := lumaProfile{
		ID:         id,
		Identity:   strings.TrimSpace(args.String("identity", "")),
		Audience:   splitProfileList(args.String("audience", "")),
		Stance:     splitProfileList(args.String("stance", "")),
		Avoid:      splitProfileList(args.String("avoid", "")),
		AssetGroup: profileDefaultAssetGroup(id),
		ExtraPath:  "profile_extra.md",
	}
	if err := validateProfile(profile); err != nil {
		return output.ErrValidation("%v", err)
	}
	extra, writeExtra, err := profileExtraFromArgs(args, "")
	if err != nil {
		return output.ErrSystem("%v", err)
	}
	if err := uploadProfileAssetsFromArgs(&profile, args); err != nil {
		return err
	}
	force := args.Has("force")
	if err := saveProfile(profile, extra, writeExtra, force); err != nil {
		return output.ErrSystem("%v", err)
	}
	if args.Has("use") {
		_ = saveProfileState(profileState{CurrentProfile: id})
	}
	if saved, err := loadProfile(profile.ID); err == nil {
		profile = saved
	}
	return printProfileResult("created", profile.ID, profile)
}

func cmdProfileUpdate(raw []string) error {
	args := cmdutil.Parse(raw)
	id := strings.TrimSpace(args.Pos(0))
	if id == "" {
		return output.ErrValidation("profile id is required")
	}
	profile, err := loadProfile(id)
	if err != nil {
		return output.ErrSystem("%v", err)
	}
	if value := strings.TrimSpace(args.String("identity", "")); value != "" {
		profile.Identity = value
	}
	if args.Has("audience") {
		profile.Audience = splitProfileList(args.String("audience", ""))
	}
	if args.Has("stance") {
		profile.Stance = splitProfileList(args.String("stance", ""))
	}
	if args.Has("avoid") {
		profile.Avoid = splitProfileList(args.String("avoid", ""))
	}
	if value := strings.TrimSpace(args.String("asset-group", "")); value != "" {
		profile.AssetGroup = value
	}
	existingExtra, _ := readProfileExtra(id)
	extra, writeExtra, err := profileExtraFromArgs(args, existingExtra)
	if err != nil {
		return output.ErrSystem("%v", err)
	}
	if args.Has("append-extra") && writeExtra {
		extra = appendProfileExtra(existingExtra, extra)
	}
	if err := uploadProfileAssetsFromArgs(&profile, args); err != nil {
		return err
	}
	if err := saveProfile(profile, extra, writeExtra, true); err != nil {
		return output.ErrSystem("%v", err)
	}
	if saved, err := loadProfile(profile.ID); err == nil {
		profile = saved
	}
	return printProfileResult("updated", profile.ID, profile)
}

func cmdProfileList(raw []string) error {
	profiles, err := listProfiles()
	if err != nil {
		return output.ErrSystem("%v", err)
	}
	state, _ := loadProfileState()
	if runtimeOpts.JSON {
		_ = output.WriteJSON(os.Stdout, output.Envelope{OK: true, Data: map[string]any{"current_profile": state.CurrentProfile, "profiles": profiles}})
		return nil
	}
	if len(profiles) == 0 {
		fmt.Println("No profiles found.")
		return nil
	}
	fmt.Printf("%-2s %-24s %-28s %s\n", "", "ID", "IDENTITY", "AUDIENCE")
	for _, item := range profiles {
		mark := " "
		if item.ID == state.CurrentProfile {
			mark = "*"
		}
		fmt.Printf("%-2s %-24s %-28s %s\n", mark, item.ID, item.Identity, strings.Join(item.Audience, ", "))
	}
	return nil
}

func cmdProfileGet(raw []string) error {
	args := cmdutil.Parse(raw)
	id := strings.TrimSpace(args.Pos(0))
	if id == "" {
		state, _ := loadProfileState()
		id = state.CurrentProfile
	}
	if id == "" {
		return output.ErrValidation("profile id is required")
	}
	view, err := loadProfileView(id)
	if err != nil {
		return output.ErrSystem("%v", err)
	}
	if runtimeOpts.JSON {
		_ = output.WriteJSON(os.Stdout, output.Envelope{OK: true, Data: view})
		return nil
	}
	data, _ := json.MarshalIndent(view.lumaProfile, "", "  ")
	fmt.Println(string(data))
	if strings.TrimSpace(view.Extra) != "" {
		fmt.Printf("\nprofile_extra.md:\n%s\n", view.Extra)
	}
	fmt.Printf("\nPath: %s\n", view.Path)
	return nil
}

func cmdProfileUse(raw []string) error {
	if len(raw) < 1 {
		return output.ErrValidation("usage: luma-cli profile use <id>")
	}
	id := strings.TrimSpace(raw[0])
	if _, err := loadProfile(id); err != nil {
		return output.ErrSystem("%v", err)
	}
	if err := saveProfileState(profileState{CurrentProfile: id}); err != nil {
		return output.ErrSystem("%v", err)
	}
	if runtimeOpts.JSON {
		_ = output.WriteJSON(os.Stdout, output.Envelope{OK: true, Data: map[string]string{"current_profile": id}})
		return nil
	}
	fmt.Printf("Current profile: %s\n", id)
	return nil
}

func cmdProfileCurrent(raw []string) error {
	state, _ := loadProfileState()
	if runtimeOpts.JSON {
		_ = output.WriteJSON(os.Stdout, output.Envelope{OK: true, Data: state})
		return nil
	}
	if state.CurrentProfile == "" {
		fmt.Println("No current profile.")
		return nil
	}
	fmt.Printf("Current profile: %s\n", state.CurrentProfile)
	return nil
}

func cmdProfileAsset(raw []string) error {
	if len(raw) < 1 {
		printProfileAssetUsage()
		return nil
	}
	switch raw[0] {
	case "upload":
		return cmdProfileAssetUpload(raw[1:])
	case "add":
		return cmdProfileAssetAdd(raw[1:])
	case "list", "ls":
		return cmdProfileAssetList(raw[1:])
	case "remove", "rm", "delete":
		return cmdProfileAssetRemove(raw[1:])
	default:
		printProfileAssetUsage()
		return nil
	}
}

func cmdProfileAssetUpload(raw []string) error {
	args := cmdutil.Parse(raw)
	profileID := strings.TrimSpace(args.String("profile", ""))
	filePath := strings.TrimSpace(args.Pos(0))
	if profileID == "" && len(args.Positionals) >= 2 {
		profileID = strings.TrimSpace(args.Pos(0))
		filePath = strings.TrimSpace(args.Pos(1))
	}
	if profileID == "" {
		state, _ := loadProfileState()
		profileID = state.CurrentProfile
	}
	if profileID == "" || filePath == "" {
		return output.ErrValidation("usage: luma-cli profile asset upload [profile_id] <file> [--kind sample|reference|brand|avatar|benchmark] [--name <display_name>]")
	}
	profile, err := loadProfile(profileID)
	if err != nil {
		return output.ErrSystem("%v", err)
	}
	asset, err := uploadOneProfileAsset(&profile, filePath, args)
	if err != nil {
		return err
	}
	if err := saveProfile(profile, "", false, true); err != nil {
		return output.ErrSystem("%v", err)
	}
	return printProfileAssetResult(profile.ID, asset)
}

func uploadProfileAssetsFromArgs(profile *lumaProfile, args cmdutil.Args) error {
	files := splitAssetFileList(args.String("asset", ""))
	files = append(files, splitAssetFileList(args.String("assets", ""))...)
	files = uniqueProfileStrings(files)
	if len(files) == 0 {
		return nil
	}
	for _, filePath := range files {
		if _, err := uploadOneProfileAsset(profile, filePath, args); err != nil {
			return err
		}
	}
	return nil
}

func uploadOneProfileAsset(profile *lumaProfile, filePath string, args cmdutil.Args) (profileAsset, error) {
	cfg, err := requireConfig()
	if err != nil {
		return profileAsset{}, output.ErrAuth("%v", err)
	}
	group := strings.TrimSpace(args.String("group", profile.AssetGroup))
	if group == "" {
		group = profileDefaultAssetGroup(profile.ID)
	}
	displayName := strings.TrimSpace(args.String("name", ""))
	kind := strings.TrimSpace(args.String("kind", "reference"))
	if displayName == "" {
		displayName = strings.TrimSuffix(filepath.Base(filePath), filepath.Ext(filePath))
	}
	objectKey, err := cloud.UploadFileWithNameAndMeta(filePath, cfg.CardKey, group, displayName, map[string]any{
		"luma_resource_kind": "profile_asset",
		"artifact_type":      "profile_asset",
		"profile_id":         profile.ID,
		"asset_kind":         kind,
	})
	if err != nil {
		return profileAsset{}, output.ErrNetwork("upload failed: %v", err)
	}
	asset := profileAsset{
		ID:        profileAssetID(objectKey, filePath),
		Kind:      kind,
		Name:      displayName,
		Group:     group,
		ObjectKey: objectKey,
		LocalPath: cleanLocalPath(filePath),
		CreatedAt: time.Now(),
	}
	profile.Assets = upsertProfileAsset(profile.Assets, asset)
	profile.AssetGroup = group
	return asset, nil
}

func cmdProfileAssetAdd(raw []string) error {
	args := cmdutil.Parse(raw)
	id := strings.TrimSpace(args.Pos(0))
	if id == "" {
		state, _ := loadProfileState()
		id = state.CurrentProfile
	}
	if id == "" {
		return output.ErrValidation("profile id is required")
	}
	objectKey := strings.TrimSpace(args.String("object-key", ""))
	if objectKey == "" {
		objectKey = strings.TrimSpace(args.String("key", ""))
	}
	if objectKey == "" {
		return output.ErrValidation("--object-key is required")
	}
	profile, err := loadProfile(id)
	if err != nil {
		return output.ErrSystem("%v", err)
	}
	group := strings.TrimSpace(args.String("group", profile.AssetGroup))
	asset := profileAsset{
		ID:        profileAssetID(objectKey, ""),
		Kind:      strings.TrimSpace(args.String("kind", "reference")),
		Name:      strings.TrimSpace(args.String("name", "")),
		Group:     group,
		ObjectKey: objectKey,
		CreatedAt: time.Now(),
	}
	profile.Assets = upsertProfileAsset(profile.Assets, asset)
	if err := saveProfile(profile, "", false, true); err != nil {
		return output.ErrSystem("%v", err)
	}
	return printProfileAssetResult(profile.ID, asset)
}

func cmdProfileAssetList(raw []string) error {
	id := ""
	if len(raw) > 0 {
		id = strings.TrimSpace(raw[0])
	}
	if id == "" {
		state, _ := loadProfileState()
		id = state.CurrentProfile
	}
	if id == "" {
		return output.ErrValidation("profile id is required")
	}
	profile, err := loadProfile(id)
	if err != nil {
		return output.ErrSystem("%v", err)
	}
	if runtimeOpts.JSON {
		_ = output.WriteJSON(os.Stdout, output.Envelope{OK: true, Data: map[string]any{"profile_id": id, "asset_group": profile.AssetGroup, "assets": profile.Assets}})
		return nil
	}
	if len(profile.Assets) == 0 {
		fmt.Println("No profile assets.")
		return nil
	}
	fmt.Printf("Assets for profile: %s\n", id)
	fmt.Printf("%-18s %-12s %-24s %s\n", "ID", "KIND", "NAME", "OBJECT_KEY")
	for _, asset := range profile.Assets {
		fmt.Printf("%-18s %-12s %-24s %s\n", asset.ID, asset.Kind, asset.Name, asset.ObjectKey)
	}
	return nil
}

func cmdProfileAssetRemove(raw []string) error {
	args := cmdutil.Parse(raw)
	id := strings.TrimSpace(args.Pos(0))
	target := strings.TrimSpace(args.Pos(1))
	if target == "" {
		target = strings.TrimSpace(args.String("id", ""))
	}
	if id == "" {
		state, _ := loadProfileState()
		id = state.CurrentProfile
		target = strings.TrimSpace(args.Pos(0))
	}
	if id == "" || target == "" {
		return output.ErrValidation("usage: luma-cli profile asset remove [profile_id] <asset_id_or_object_key>")
	}
	profile, err := loadProfile(id)
	if err != nil {
		return output.ErrSystem("%v", err)
	}
	assets := make([]profileAsset, 0, len(profile.Assets))
	removed := false
	for _, asset := range profile.Assets {
		if asset.ID == target || asset.ObjectKey == target {
			removed = true
			continue
		}
		assets = append(assets, asset)
	}
	if !removed {
		return output.ErrValidation("asset not found: %s", target)
	}
	profile.Assets = assets
	if err := saveProfile(profile, "", false, true); err != nil {
		return output.ErrSystem("%v", err)
	}
	if runtimeOpts.JSON {
		_ = output.WriteJSON(os.Stdout, output.Envelope{OK: true, Data: map[string]any{"profile_id": id, "removed": target}})
		return nil
	}
	fmt.Printf("Removed profile asset: %s\n", target)
	return nil
}

func profileStatePath() (string, error) {
	dir, err := appconfig.Dir()
	if err != nil {
		return "", err
	}
	return filepath.Join(dir, "profile_state.json"), nil
}

func validateProfileID(id string) error {
	if strings.TrimSpace(id) == "" {
		return fmt.Errorf("profile id is required")
	}
	if !profileIDPattern.MatchString(id) {
		return fmt.Errorf("invalid profile id %q; use letters, numbers, dot, underscore, or dash", id)
	}
	return nil
}

func validateProfile(profile lumaProfile) error {
	if err := validateProfileID(profile.ID); err != nil {
		return err
	}
	if strings.TrimSpace(profile.Identity) == "" {
		return fmt.Errorf("identity is required")
	}
	if len(cleanProfileList(profile.Audience)) == 0 {
		return fmt.Errorf("audience is required")
	}
	if len(cleanProfileList(profile.Stance)) == 0 {
		return fmt.Errorf("stance is required")
	}
	if len(cleanProfileList(profile.Avoid)) == 0 {
		return fmt.Errorf("avoid is required")
	}
	return nil
}

func saveProfile(profile lumaProfile, extra string, writeExtra bool, overwrite bool) error {
	if err := validateProfile(profile); err != nil {
		return err
	}
	if !overwrite {
		if _, err := loadProfile(profile.ID); err == nil {
			return fmt.Errorf("profile already exists: %s", profile.ID)
		}
	}
	now := time.Now()
	existing, err := loadProfile(profile.ID)
	if err == nil && !existing.CreatedAt.IsZero() {
		profile.CreatedAt = existing.CreatedAt
	} else if profile.CreatedAt.IsZero() {
		profile.CreatedAt = now
	}
	profile.UpdatedAt = now
	profile.Audience = cleanProfileList(profile.Audience)
	profile.Stance = cleanProfileList(profile.Stance)
	profile.Avoid = cleanProfileList(profile.Avoid)
	if profile.AssetGroup == "" {
		profile.AssetGroup = profileDefaultAssetGroup(profile.ID)
	}
	profile.ExtraPath = "profile_extra.md"
	cfg, err := requireConfig()
	if err != nil {
		return err
	}
	meta := map[string]any{
		"luma_resource_kind": "profile",
		"artifact_type":      "profile",
		"profile_id":         profile.ID,
		"is_current":         true,
	}
	if err := uploadJSONResource(profileDefaultAssetGroup(profile.ID), "profile.current.json", profile, cfg.CardKey, meta); err != nil {
		return err
	}
	if writeExtra {
		extraMeta := map[string]any{
			"luma_resource_kind": "profile_extra",
			"artifact_type":      "profile_extra",
			"profile_id":         profile.ID,
			"is_current":         true,
		}
		if err := uploadTextResource(profileDefaultAssetGroup(profile.ID), "profile_extra.current.md", extra, cfg.CardKey, extraMeta); err != nil {
			return err
		}
	}
	return nil
}

func loadProfile(id string) (lumaProfile, error) {
	if err := validateProfileID(id); err != nil {
		return lumaProfile{}, err
	}
	cfg, err := requireConfig()
	if err != nil {
		return lumaProfile{}, err
	}
	item, err := latestResourceByDisplayName(profileDefaultAssetGroup(id), "profile.current", cfg.CardKey)
	if err != nil {
		return lumaProfile{}, err
	}
	text, err := downloadResourceText(strAny(item["resource_url"]))
	if err != nil {
		return lumaProfile{}, err
	}
	var profile lumaProfile
	if err := json.Unmarshal([]byte(text), &profile); err != nil {
		return lumaProfile{}, err
	}
	if profile.AssetGroup == "" {
		profile.AssetGroup = profileDefaultAssetGroup(profile.ID)
	}
	return profile, nil
}

func loadProfileView(id string) (profileView, error) {
	profile, err := loadProfile(id)
	if err != nil {
		return profileView{}, err
	}
	extra, err := readProfileExtra(id)
	if err != nil {
		return profileView{}, err
	}
	return profileView{lumaProfile: profile, Extra: extra, Path: "cloud:" + profileDefaultAssetGroup(id)}, nil
}

func readProfileExtra(id string) (string, error) {
	if err := validateProfileID(id); err != nil {
		return "", err
	}
	cfg, err := requireConfig()
	if err != nil {
		return "", err
	}
	item, err := latestResourceByDisplayName(profileDefaultAssetGroup(id), "profile_extra.current", cfg.CardKey)
	if err != nil {
		return "", nil
	}
	return downloadResourceText(strAny(item["resource_url"]))
}

func listProfiles() ([]lumaProfile, error) {
	cfg, err := requireConfig()
	if err != nil {
		return nil, err
	}
	items, err := cloud.AssetList("", cfg.CardKey)
	if err != nil {
		return nil, err
	}
	profiles := []lumaProfile{}
	seen := map[string]bool{}
	for _, raw := range items {
		item := mapFromAny(raw)
		meta := mapFromAny(item["meta"])
		filename := strAny(item["filename"])
		stem := strings.TrimSuffix(filename, filepath.Ext(filename))
		if !(strAny(meta["luma_resource_kind"]) == "profile" || strings.HasPrefix(stem, "profile.current_") || stem == "profile.current") {
			continue
		}
		if seen[strAny(item["object_key"])] {
			continue
		}
		text, err := downloadResourceText(strAny(item["resource_url"]))
		if err != nil {
			continue
		}
		var profile lumaProfile
		if err := json.Unmarshal([]byte(text), &profile); err != nil {
			continue
		}
		if profile.ID == "" || seen[profile.ID] {
			continue
		}
		if profile.AssetGroup == "" {
			profile.AssetGroup = profileDefaultAssetGroup(profile.ID)
		}
		seen[profile.ID] = true
		seen[strAny(item["object_key"])] = true
		profiles = append(profiles, profile)
	}
	sort.Slice(profiles, func(i, j int) bool {
		return profiles[i].UpdatedAt.After(profiles[j].UpdatedAt)
	})
	return profiles, nil
}

func loadProfileState() (profileState, error) {
	path, err := profileStatePath()
	if err != nil {
		return profileState{}, err
	}
	var state profileState
	if err := readJSONStruct(path, &state); err != nil {
		if errors.Is(err, os.ErrNotExist) {
			return profileState{}, nil
		}
		return profileState{}, err
	}
	return state, nil
}

func saveProfileState(state profileState) error {
	path, err := profileStatePath()
	if err != nil {
		return err
	}
	return writeJSONFile(path, state)
}

func readJSONStruct(path string, target any) error {
	data, err := os.ReadFile(path)
	if err != nil {
		return err
	}
	return json.Unmarshal(data, target)
}

func uploadJSONResource(groupName, resourceName string, value any, cardKey string, meta map[string]any) error {
	data, err := json.MarshalIndent(value, "", "  ")
	if err != nil {
		return err
	}
	return uploadTextResource(groupName, resourceName, string(data), cardKey, meta)
}

func uploadTextResource(groupName, resourceName, text, cardKey string, meta map[string]any) error {
	ext := filepath.Ext(resourceName)
	if ext == "" {
		ext = ".txt"
	}
	tmp, err := os.CreateTemp("", "luma-resource-*"+ext)
	if err != nil {
		return err
	}
	tmpPath := tmp.Name()
	defer os.Remove(tmpPath)
	if _, err := tmp.WriteString(text); err != nil {
		tmp.Close()
		return err
	}
	if err := tmp.Close(); err != nil {
		return err
	}
	_, err = cloud.UploadFileWithNameAndMeta(tmpPath, cardKey, groupName, resourceName, meta)
	return err
}

func latestResourceByDisplayName(groupName, displayName, cardKey string) (map[string]any, error) {
	items, err := cloud.AssetList(groupName, cardKey)
	if err != nil {
		return nil, err
	}
	for _, raw := range items {
		item := mapFromAny(raw)
		meta := mapFromAny(item["meta"])
		filename := strAny(item["filename"])
		stem := strings.TrimSuffix(filename, filepath.Ext(filename))
		if strAny(meta["display_name"]) == displayName || stem == displayName || strings.HasPrefix(stem, displayName+"_") {
			if strAny(item["resource_url"]) == "" {
				return nil, fmt.Errorf("resource %s has no signed URL", displayName)
			}
			return item, nil
		}
	}
	return nil, fmt.Errorf("cloud resource not found: %s/%s", groupName, displayName)
}

func resourceByObjectKey(groupName, objectKey, cardKey string) (map[string]any, error) {
	objectKey = strings.TrimSpace(objectKey)
	if objectKey == "" {
		return nil, fmt.Errorf("object key is required")
	}
	items, err := cloud.AssetList(groupName, cardKey)
	if err != nil {
		return nil, err
	}
	for _, raw := range items {
		item := mapFromAny(raw)
		if strAny(item["object_key"]) == objectKey {
			if strAny(item["resource_url"]) == "" {
				return nil, fmt.Errorf("resource %s has no signed URL", objectKey)
			}
			return item, nil
		}
	}
	return nil, fmt.Errorf("cloud resource not found: %s", objectKey)
}

func downloadResourceText(resourceURL string) (string, error) {
	resourceURL = strings.TrimSpace(resourceURL)
	if resourceURL == "" {
		return "", fmt.Errorf("resource URL is empty")
	}
	tmp, err := os.CreateTemp("", "luma-resource-download-*")
	if err != nil {
		return "", err
	}
	tmpPath := tmp.Name()
	_ = tmp.Close()
	defer os.Remove(tmpPath)
	if err := atom.DownloadFile(resourceURL, tmpPath); err != nil {
		return "", err
	}
	data, err := os.ReadFile(tmpPath)
	if err != nil {
		return "", err
	}
	return string(data), nil
}

func mapFromAny(value any) map[string]any {
	if value == nil {
		return map[string]any{}
	}
	if item, ok := value.(map[string]any); ok {
		return item
	}
	data, err := json.Marshal(value)
	if err != nil {
		return map[string]any{}
	}
	var out map[string]any
	if err := json.Unmarshal(data, &out); err != nil {
		return map[string]any{}
	}
	return out
}

func profileExtraFromArgs(args cmdutil.Args, existing string) (string, bool, error) {
	extra := strings.TrimSpace(args.String("extra", ""))
	if filePath := strings.TrimSpace(args.String("extra-file", "")); filePath != "" {
		data, err := os.ReadFile(filePath)
		if err != nil {
			return "", false, fmt.Errorf("read extra file failed: %w", err)
		}
		extra = strings.TrimSpace(string(data))
	}
	if args.Has("no-extra") {
		return "", true, nil
	}
	if extra != "" || args.Has("extra") || args.Has("extra-file") {
		return extra, true, nil
	}
	return existing, false, nil
}

func appendProfileExtra(existing, extra string) string {
	existing = strings.TrimRight(existing, "\r\n")
	extra = strings.TrimSpace(extra)
	if existing == "" {
		return extra
	}
	if extra == "" {
		return existing + "\n"
	}
	return existing + "\n\n" + extra + "\n"
}

func splitProfileList(value string) []string {
	return cleanProfileList(strings.FieldsFunc(value, func(r rune) bool {
		return r == ',' || r == ';' || r == '；' || r == '\n' || r == '\t'
	}))
}

func splitAssetFileList(value string) []string {
	parts := strings.FieldsFunc(value, func(r rune) bool {
		return r == ',' || r == '，' || r == ';' || r == '；' || r == '\n' || r == '\t'
	})
	out := make([]string, 0, len(parts))
	seen := map[string]bool{}
	for _, part := range parts {
		item := strings.TrimSpace(part)
		if item == "" || seen[item] {
			continue
		}
		seen[item] = true
		out = append(out, item)
	}
	return out
}

func cleanProfileList(values []string) []string {
	out := make([]string, 0, len(values))
	seen := map[string]bool{}
	for _, value := range values {
		for _, part := range strings.FieldsFunc(value, func(r rune) bool { return r == ',' || r == ';' || r == '；' || r == '\n' || r == '\t' }) {
			item := strings.TrimSpace(part)
			if item == "" || seen[item] {
				continue
			}
			seen[item] = true
			out = append(out, item)
		}
	}
	return out
}

func uniqueProfileStrings(values []string) []string {
	out := make([]string, 0, len(values))
	seen := map[string]bool{}
	for _, value := range values {
		item := strings.TrimSpace(value)
		if item == "" || seen[item] {
			continue
		}
		seen[item] = true
		out = append(out, item)
	}
	return out
}

func profileDefaultAssetGroup(id string) string {
	return "profile_" + slugForID(id)
}

func profileAssetID(objectKey, localPath string) string {
	base := firstNonEmpty(objectKey, localPath, fmt.Sprint(time.Now().UnixNano()))
	slug := slugForID(base)
	if len(slug) > 16 {
		slug = slug[:16]
	}
	return "asset_" + slug
}

func cleanLocalPath(path string) string {
	abs, err := filepath.Abs(path)
	if err != nil {
		return path
	}
	return abs
}

func upsertProfileAsset(items []profileAsset, asset profileAsset) []profileAsset {
	for i := range items {
		if items[i].ID == asset.ID || (asset.ObjectKey != "" && items[i].ObjectKey == asset.ObjectKey) {
			items[i] = asset
			return items
		}
	}
	return append(items, asset)
}

func printProfileResult(action, id string, profile lumaProfile) error {
	if runtimeOpts.JSON {
		_ = output.WriteJSON(os.Stdout, output.Envelope{OK: true, Data: profile})
		return nil
	}
	fmt.Printf("Profile %s: %s\n", action, id)
	fmt.Printf("Asset group: %s\n", profile.AssetGroup)
	return nil
}

func printProfileAssetResult(profileID string, asset profileAsset) error {
	if runtimeOpts.JSON {
		_ = output.WriteJSON(os.Stdout, output.Envelope{OK: true, Data: map[string]any{"profile_id": profileID, "asset": asset}})
		return nil
	}
	fmt.Printf("Profile asset saved: %s\n", asset.ID)
	fmt.Printf("Profile: %s\n", profileID)
	if asset.Name != "" {
		fmt.Printf("Name: %s\n", asset.Name)
	}
	fmt.Printf("Group: %s\n", asset.Group)
	if asset.ObjectKey != "" {
		fmt.Printf("Object key: %s\n", asset.ObjectKey)
	}
	return nil
}

func printProfileUsage() {
	fmt.Println("luma-cli profile <subcommand>")
	fmt.Println("  create <id> --identity <text> --audience <list> --stance <list> --avoid <list> [--asset <file>]")
	fmt.Println("  update <id> [--identity <text>] [--audience <list>] [--stance <list>] [--avoid <list>] [--asset <file>]")
	fmt.Println("  list")
	fmt.Println("  get [id]")
	fmt.Println("  use <id>")
	fmt.Println("  current")
	fmt.Println("  asset <subcommand>")
}

func printProfileAssetUsage() {
	fmt.Println("luma-cli profile asset <subcommand>")
	fmt.Println("  upload [profile_id] <file> [--kind reference|benchmark] [--name <display_name>]")
	fmt.Println("  add [profile_id] --object-key <key> [--group <group>] [--kind reference] [--name <name>]")
	fmt.Println("  list [profile_id]")
	fmt.Println("  remove [profile_id] <asset_id_or_object_key>")
}
