package cloud

import "net/url"

type AvatarPersonaRef struct {
	AssetID     string   `json:"asset_id"`
	Usage       []string `json:"usage,omitempty"`
	DisplayName string   `json:"display_name,omitempty"`
}

type AvatarPersona struct {
	SchemaVersion       string             `json:"schema_version,omitempty"`
	AvatarPersonaID     string             `json:"avatar_persona_id,omitempty"`
	AvatarName          string             `json:"avatar_name,omitempty"`
	AvatarImageAssetID  string             `json:"avatar_image_asset_id,omitempty"`
	PreviewImageAssetID string             `json:"preview_image_asset_id,omitempty"`
	CreatedByUID        string             `json:"created_by_uid,omitempty"`
	Visibility          string             `json:"visibility,omitempty"`
	Status              string             `json:"status,omitempty"`
	RoleDescription     string             `json:"role_description,omitempty"`
	Audience            string             `json:"audience,omitempty"`
	Voices              []AvatarPersonaRef `json:"voices,omitempty"`
	Roles               []AvatarPersonaRef `json:"roles,omitempty"`
	MissingRequirements []string           `json:"missing_requirements,omitempty"`
	CreatedAt           string             `json:"created_at,omitempty"`
	UpdatedAt           string             `json:"updated_at,omitempty"`
}

type AvatarPersonaListResponse struct {
	Items []AvatarPersona `json:"items"`
}

type AvatarPersonaOptionsResponse struct {
	Voices []AssetItem `json:"voices"`
	Roles  []AssetItem `json:"roles"`
	Images []AssetItem `json:"images"`
}

type AvatarPersonaValidateResponse struct {
	OK                  bool          `json:"ok"`
	Status              string        `json:"status"`
	MissingRequirements []string      `json:"missing_requirements"`
	Persona             AvatarPersona `json:"persona"`
}

func AvatarPersonasCreate(payload map[string]any, cardKey string) (*AvatarPersona, error) {
	result, err := apiRequest("POST", "/api/avatar-personas", payload, cardKey)
	if err != nil {
		return nil, err
	}
	item, err := mapToStruct[AvatarPersona](result)
	return &item, err
}

func AvatarPersonasList(cardKey string) (*AvatarPersonaListResponse, error) {
	result, err := apiRequest("GET", "/api/avatar-personas", nil, cardKey)
	if err != nil {
		return nil, err
	}
	item, err := mapToStruct[AvatarPersonaListResponse](result)
	return &item, err
}

func AvatarPersonasGet(id, cardKey string) (*AvatarPersona, error) {
	result, err := apiRequest("GET", "/api/avatar-personas/"+url.PathEscape(id), nil, cardKey)
	if err != nil {
		return nil, err
	}
	item, err := mapToStruct[AvatarPersona](result)
	return &item, err
}

func AvatarPersonasUpdate(id string, payload map[string]any, cardKey string) (*AvatarPersona, error) {
	result, err := apiRequest("PATCH", "/api/avatar-personas/"+url.PathEscape(id), payload, cardKey)
	if err != nil {
		return nil, err
	}
	item, err := mapToStruct[AvatarPersona](result)
	return &item, err
}

func AvatarPersonasDelete(id, cardKey string) (map[string]any, error) {
	return apiRequest("DELETE", "/api/avatar-personas/"+url.PathEscape(id), nil, cardKey)
}

func AvatarPersonasOptions(cardKey string) (*AvatarPersonaOptionsResponse, error) {
	result, err := apiRequest("GET", "/api/avatar-personas/options", nil, cardKey)
	if err != nil {
		return nil, err
	}
	item, err := mapToStruct[AvatarPersonaOptionsResponse](result)
	return &item, err
}

func AvatarPersonasValidate(payload map[string]any, cardKey string) (*AvatarPersonaValidateResponse, error) {
	result, err := apiRequest("POST", "/api/avatar-personas/validate", payload, cardKey)
	if err != nil {
		return nil, err
	}
	item, err := mapToStruct[AvatarPersonaValidateResponse](result)
	return &item, err
}

func AvatarPersonasBindVoice(id string, payload map[string]any, cardKey string) (*AvatarPersona, error) {
	result, err := apiRequest("POST", "/api/avatar-personas/"+url.PathEscape(id)+"/voices", payload, cardKey)
	if err != nil {
		return nil, err
	}
	item, err := mapToStruct[AvatarPersona](result)
	return &item, err
}

func AvatarPersonasUnbindVoice(id, assetID, cardKey string) (*AvatarPersona, error) {
	result, err := apiRequest("DELETE", "/api/avatar-personas/"+url.PathEscape(id)+"/voices/"+url.PathEscape(assetID), nil, cardKey)
	if err != nil {
		return nil, err
	}
	item, err := mapToStruct[AvatarPersona](result)
	return &item, err
}

func AvatarPersonasBindRole(id string, payload map[string]any, cardKey string) (*AvatarPersona, error) {
	result, err := apiRequest("POST", "/api/avatar-personas/"+url.PathEscape(id)+"/roles", payload, cardKey)
	if err != nil {
		return nil, err
	}
	item, err := mapToStruct[AvatarPersona](result)
	return &item, err
}

func AvatarPersonasUnbindRole(id, assetID, cardKey string) (*AvatarPersona, error) {
	result, err := apiRequest("DELETE", "/api/avatar-personas/"+url.PathEscape(id)+"/roles/"+url.PathEscape(assetID), nil, cardKey)
	if err != nil {
		return nil, err
	}
	item, err := mapToStruct[AvatarPersona](result)
	return &item, err
}
