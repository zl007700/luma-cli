package commands

import "sort"

type commandHandler func(args []string) error

type commandSpec struct {
	Name        string
	Description string
	Handler     commandHandler
}

func commandRegistry() map[string]commandSpec {
	specs := []commandSpec{
		{Name: "agent", Description: "Backend-owned agent abilities", Handler: cmdAgent},
		{Name: "align", Description: "Align subtitle segments to audio via cloud API", Handler: cmdAlign},
		{Name: "auth", Description: "Authentication commands", Handler: cmdAuth},
		{Name: "asr", Description: "Speech recognition", Handler: cmdASR},
		{Name: "asset", Description: "Asset upload and listing", Handler: cmdAsset},
		{Name: "bgm", Description: "Background music mixing", Handler: cmdBGM},
		{Name: "content", Description: "Content planning and topic mining", Handler: cmdContent},
		{Name: "cover", Description: "Cover frame and image rendering", Handler: cmdCover},
		{Name: "defaults", Description: "Show product default settings", Handler: cmdDefaults},
		{Name: "download", Description: "Download a remote file", Handler: cmdDownload},
		{Name: "douyin", Description: "Douyin helpers", Handler: cmdDouyin},
		{Name: "enhance", Description: "Video enhancement", Handler: cmdEnhance},
		{Name: "env", Description: "Backend environment", Handler: cmdEnv},
		{Name: "image", Description: "AI image generation", Handler: cmdImage},
		{Name: "lipsync", Description: "Digital human lip sync", Handler: cmdLipSync},
		{Name: "material", Description: "Local material description for PIP planning", Handler: cmdMaterial},
		{Name: "pip", Description: "Picture-in-picture rendering", Handler: cmdPIP},
		{Name: "profile", Description: "Global content profile management", Handler: cmdProfile},
		{Name: "project", Description: "Project workspace commands", Handler: cmdProject},
		{Name: "research", Description: "Content research and persona helpers", Handler: cmdResearch},
		{Name: "resource", Description: "Cloud-managed client resources", Handler: cmdResource},
		{Name: "runtime", Description: "Local runtime installation", Handler: cmdRuntime},
		{Name: "script", Description: "Cloud script generation helpers", Handler: cmdScript},
		{Name: "skills", Description: "Install and sync agent skills", Handler: cmdSkills},
		{Name: "social", Description: "Social platform video download (Douyin)", Handler: cmdSocial},
		{Name: "subtitle", Description: "Subtitle generation and rendering", Handler: cmdSubtitle},
		{Name: "task", Description: "Cloud task status", Handler: cmdTask},
		{Name: "tools", Description: "Agent tool discovery", Handler: cmdTools},
		{Name: "tts", Description: "Text to speech", Handler: cmdTTS},
		{Name: "update", Description: "Update CLI and sync skills", Handler: cmdUpdate},
		{Name: "video", Description: "AI video generation", Handler: cmdVideo},
		{Name: "viral", Description: "Viral copy helpers", Handler: cmdViral},
		{Name: "voice", Description: "Voice clone and listing", Handler: cmdVoice},
	}

	registry := make(map[string]commandSpec, len(specs))
	for _, spec := range specs {
		registry[spec.Name] = spec
	}
	return registry
}

func commandNames() []string {
	registry := commandRegistry()
	names := make([]string, 0, len(registry))
	for name := range registry {
		names = append(names, name)
	}
	sort.Strings(names)
	return names
}
