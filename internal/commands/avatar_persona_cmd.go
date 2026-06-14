package commands

import (
	"encoding/json"
	"fmt"
	"os"
	"strings"

	"github.com/luma-cli/lumer-cli/cloud"
	"github.com/luma-cli/lumer-cli/internal/cmdutil"
	"github.com/luma-cli/lumer-cli/internal/output"
)

func cmdAvatarPersona(args []string) error {
	if len(args) < 1 {
		printAvatarPersonaUsage()
		return nil
	}
	switch args[0] {
	case "create":
		return cmdAvatarPersonaCreate(args[1:])
	case "list", "ls":
		return cmdAvatarPersonaList(args[1:])
	case "get", "show":
		return cmdAvatarPersonaGet(args[1:])
	case "update":
		return cmdAvatarPersonaUpdate(args[1:])
	case "delete", "remove", "rm":
		return cmdAvatarPersonaDelete(args[1:])
	case "options":
		return cmdAvatarPersonaOptions(args[1:])
	case "validate":
		return cmdAvatarPersonaValidate(args[1:])
	case "bind-voice":
		return cmdAvatarPersonaBindVoice(args[1:])
	case "unbind-voice":
		return cmdAvatarPersonaUnbindVoice(args[1:])
	case "bind-role":
		return cmdAvatarPersonaBindRole(args[1:])
	case "unbind-role":
		return cmdAvatarPersonaUnbindRole(args[1:])
	default:
		fmt.Printf("unknown avatar-persona subcommand: %s\n\n", args[0])
		printAvatarPersonaUsage()
		return nil
	}
}

func cmdAvatarPersonaCreate(raw []string) error {
	parsed := cmdutil.Parse(raw)
	payload, err := avatarPersonaPayloadFromArgs(parsed, false)
	if err != nil {
		return err
	}
	cfg, err := requireConfig()
	if err != nil {
		return err
	}
	item, err := cloud.AvatarPersonasCreate(payload, cfg.CardKey)
	if err != nil {
		return output.ErrNetwork("avatar persona create failed: %v", err)
	}
	return printAvatarPersona(item)
}

func cmdAvatarPersonaList(raw []string) error {
	cfg, err := requireConfig()
	if err != nil {
		return err
	}
	result, err := cloud.AvatarPersonasList(cfg.CardKey)
	if err != nil {
		return output.ErrNetwork("avatar persona list failed: %v", err)
	}
	if runtimeOpts.JSON {
		_ = output.WriteJSON(os.Stdout, output.Envelope{OK: true, Data: result})
		return nil
	}
	if len(result.Items) == 0 {
		fmt.Println("No avatar personas found.")
		return nil
	}
	fmt.Printf("%-34s %-20s %-10s %-10s %s\n", "ID", "NAME", "STATUS", "VISIBILITY", "MISSING")
	for _, item := range result.Items {
		fmt.Printf("%-34s %-20s %-10s %-10s %s\n", item.AvatarPersonaID, item.AvatarName, item.Status, item.Visibility, strings.Join(item.MissingRequirements, ","))
	}
	return nil
}

func cmdAvatarPersonaGet(raw []string) error {
	id := strings.TrimSpace(cmdutil.Parse(raw).Pos(0))
	if id == "" {
		return output.ErrValidation("usage: luma-cli avatar-persona get <avatar_persona_id>")
	}
	cfg, err := requireConfig()
	if err != nil {
		return err
	}
	item, err := cloud.AvatarPersonasGet(id, cfg.CardKey)
	if err != nil {
		return output.ErrNetwork("avatar persona get failed: %v", err)
	}
	return printAvatarPersona(item)
}

func cmdAvatarPersonaUpdate(raw []string) error {
	parsed := cmdutil.Parse(raw)
	id := strings.TrimSpace(parsed.Pos(0))
	if id == "" {
		return output.ErrValidation("usage: luma-cli avatar-persona update <avatar_persona_id> [--name ...]")
	}
	payload, err := avatarPersonaPayloadFromArgs(parsed, true)
	if err != nil {
		return err
	}
	cfg, err := requireConfig()
	if err != nil {
		return err
	}
	item, err := cloud.AvatarPersonasUpdate(id, payload, cfg.CardKey)
	if err != nil {
		return output.ErrNetwork("avatar persona update failed: %v", err)
	}
	return printAvatarPersona(item)
}

func cmdAvatarPersonaDelete(raw []string) error {
	id := strings.TrimSpace(cmdutil.Parse(raw).Pos(0))
	if id == "" {
		return output.ErrValidation("usage: luma-cli avatar-persona delete <avatar_persona_id>")
	}
	cfg, err := requireConfig()
	if err != nil {
		return err
	}
	result, err := cloud.AvatarPersonasDelete(id, cfg.CardKey)
	if err != nil {
		return output.ErrNetwork("avatar persona delete failed: %v", err)
	}
	if runtimeOpts.JSON {
		_ = output.WriteJSON(os.Stdout, output.Envelope{OK: true, Data: result})
		return nil
	}
	fmt.Printf("Deleted avatar persona: %s\n", id)
	return nil
}

func cmdAvatarPersonaOptions(raw []string) error {
	cfg, err := requireConfig()
	if err != nil {
		return err
	}
	result, err := cloud.AvatarPersonasOptions(cfg.CardKey)
	if err != nil {
		return output.ErrNetwork("avatar persona options failed: %v", err)
	}
	if runtimeOpts.JSON {
		_ = output.WriteJSON(os.Stdout, output.Envelope{OK: true, Data: result})
		return nil
	}
	fmt.Printf("Voices: %d\n", len(result.Voices))
	fmt.Printf("Roles: %d\n", len(result.Roles))
	fmt.Printf("Images: %d\n", len(result.Images))
	return nil
}

func cmdAvatarPersonaValidate(raw []string) error {
	parsed := cmdutil.Parse(raw)
	payload, err := avatarPersonaPayloadFromArgs(parsed, false)
	if err != nil {
		return err
	}
	cfg, err := requireConfig()
	if err != nil {
		return err
	}
	result, err := cloud.AvatarPersonasValidate(payload, cfg.CardKey)
	if err != nil {
		return output.ErrNetwork("avatar persona validate failed: %v", err)
	}
	if runtimeOpts.JSON {
		_ = output.WriteJSON(os.Stdout, output.Envelope{OK: true, Data: result})
		return nil
	}
	fmt.Printf("Valid: %v\n", result.OK)
	fmt.Printf("Status: %s\n", result.Status)
	if len(result.MissingRequirements) > 0 {
		fmt.Printf("Missing: %s\n", strings.Join(result.MissingRequirements, ", "))
	}
	return nil
}

func cmdAvatarPersonaBindVoice(raw []string) error {
	parsed := cmdutil.Parse(raw)
	id := strings.TrimSpace(parsed.Pos(0))
	assetID := strings.TrimSpace(parsed.Pos(1))
	if id == "" || assetID == "" {
		return output.ErrValidation("usage: luma-cli avatar-persona bind-voice <avatar_persona_id> <voice_asset_id> [--usage default_tts]")
	}
	return avatarPersonaBind(id, assetID, parsed, true)
}

func cmdAvatarPersonaUnbindVoice(raw []string) error {
	parsed := cmdutil.Parse(raw)
	id := strings.TrimSpace(parsed.Pos(0))
	assetID := strings.TrimSpace(parsed.Pos(1))
	if id == "" || assetID == "" {
		return output.ErrValidation("usage: luma-cli avatar-persona unbind-voice <avatar_persona_id> <voice_asset_id>")
	}
	cfg, err := requireConfig()
	if err != nil {
		return err
	}
	item, err := cloud.AvatarPersonasUnbindVoice(id, assetID, cfg.CardKey)
	if err != nil {
		return output.ErrNetwork("avatar persona unbind voice failed: %v", err)
	}
	return printAvatarPersona(item)
}

func cmdAvatarPersonaBindRole(raw []string) error {
	parsed := cmdutil.Parse(raw)
	id := strings.TrimSpace(parsed.Pos(0))
	assetID := strings.TrimSpace(parsed.Pos(1))
	if id == "" || assetID == "" {
		return output.ErrValidation("usage: luma-cli avatar-persona bind-role <avatar_persona_id> <role_asset_id> [--usage default_lipsync]")
	}
	return avatarPersonaBind(id, assetID, parsed, false)
}

func cmdAvatarPersonaUnbindRole(raw []string) error {
	parsed := cmdutil.Parse(raw)
	id := strings.TrimSpace(parsed.Pos(0))
	assetID := strings.TrimSpace(parsed.Pos(1))
	if id == "" || assetID == "" {
		return output.ErrValidation("usage: luma-cli avatar-persona unbind-role <avatar_persona_id> <role_asset_id>")
	}
	cfg, err := requireConfig()
	if err != nil {
		return err
	}
	item, err := cloud.AvatarPersonasUnbindRole(id, assetID, cfg.CardKey)
	if err != nil {
		return output.ErrNetwork("avatar persona unbind role failed: %v", err)
	}
	return printAvatarPersona(item)
}

func avatarPersonaBind(id, assetID string, parsed cmdutil.Args, voice bool) error {
	cfg, err := requireConfig()
	if err != nil {
		return err
	}
	payload := map[string]any{
		"asset_id": assetID,
		"usage":    splitAvatarPersonaList(parsed.String("usage", "")),
	}
	if displayName := strings.TrimSpace(parsed.String("name", parsed.String("display-name", ""))); displayName != "" {
		payload["display_name"] = displayName
	}
	var item *cloud.AvatarPersona
	if voice {
		item, err = cloud.AvatarPersonasBindVoice(id, payload, cfg.CardKey)
	} else {
		item, err = cloud.AvatarPersonasBindRole(id, payload, cfg.CardKey)
	}
	if err != nil {
		return output.ErrNetwork("avatar persona bind failed: %v", err)
	}
	return printAvatarPersona(item)
}

func avatarPersonaPayloadFromArgs(parsed cmdutil.Args, update bool) (map[string]any, error) {
	payload := map[string]any{}
	if filePath := strings.TrimSpace(parsed.String("file", "")); filePath != "" {
		data, err := os.ReadFile(filePath)
		if err != nil {
			return nil, output.ErrSystem("read file %s: %v", filePath, err)
		}
		if err := json.Unmarshal(data, &payload); err != nil {
			return nil, output.ErrValidation("--file must contain a JSON object: %v", err)
		}
	}
	setString := func(flag, key string) {
		if parsed.Has(flag) {
			payload[key] = strings.TrimSpace(parsed.String(flag, ""))
		}
	}
	setString("name", "avatar_name")
	setString("avatar-name", "avatar_name")
	setString("role-description", "role_description")
	setString("audience", "audience")
	setString("visibility", "visibility")
	setString("status", "status")
	setString("avatar-image", "avatar_image_asset_id")
	setString("preview-image", "preview_image_asset_id")
	if parsed.Has("voice") {
		payload["voices"] = []map[string]any{{"asset_id": strings.TrimSpace(parsed.String("voice", "")), "usage": []string{"default_tts"}}}
	}
	if parsed.Has("role") {
		payload["roles"] = []map[string]any{{"asset_id": strings.TrimSpace(parsed.String("role", "")), "usage": []string{"default_lipsync"}}}
	}
	currentName, _ := payload["avatar_name"].(string)
	if !update && strings.TrimSpace(currentName) == "" && strings.TrimSpace(parsed.Pos(0)) != "" {
		payload["avatar_name"] = strings.TrimSpace(parsed.Pos(0))
	}
	return payload, nil
}

func splitAvatarPersonaList(value string) []string {
	parts := strings.FieldsFunc(value, func(r rune) bool {
		return r == ',' || r == ';' || r == '\n' || r == '，' || r == '；'
	})
	out := []string{}
	seen := map[string]bool{}
	for _, part := range parts {
		text := strings.TrimSpace(part)
		if text != "" && !seen[text] {
			out = append(out, text)
			seen[text] = true
		}
	}
	return out
}

func printAvatarPersona(item *cloud.AvatarPersona) error {
	if runtimeOpts.JSON {
		_ = output.WriteJSON(os.Stdout, output.Envelope{OK: true, Data: item})
		return nil
	}
	fmt.Printf("Avatar persona: %s\n", item.AvatarPersonaID)
	fmt.Printf("Name: %s\n", item.AvatarName)
	fmt.Printf("Status: %s\n", item.Status)
	fmt.Printf("Visibility: %s\n", item.Visibility)
	if len(item.MissingRequirements) > 0 {
		fmt.Printf("Missing: %s\n", strings.Join(item.MissingRequirements, ", "))
	}
	fmt.Printf("Voices: %d\n", len(item.Voices))
	fmt.Printf("Roles: %d\n", len(item.Roles))
	return nil
}

func printAvatarPersonaUsage() {
	fmt.Println("luma-cli avatar-persona <subcommand>")
	fmt.Println("  create [name] --role-description <text> --audience <text> [--voice <asset_id>] [--role <asset_id>]")
	fmt.Println("  list")
	fmt.Println("  get <avatar_persona_id>")
	fmt.Println("  update <avatar_persona_id> [--name <text>] [--role-description <text>] [--audience <text>]")
	fmt.Println("  delete <avatar_persona_id>")
	fmt.Println("  options")
	fmt.Println("  validate [--file persona.json]")
	fmt.Println("  bind-voice <avatar_persona_id> <voice_asset_id> [--usage default_tts]")
	fmt.Println("  unbind-voice <avatar_persona_id> <voice_asset_id>")
	fmt.Println("  bind-role <avatar_persona_id> <role_asset_id> [--usage default_lipsync]")
	fmt.Println("  unbind-role <avatar_persona_id> <role_asset_id>")
}
