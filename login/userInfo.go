package login

import (
	"bytes"
	"errors"
	"fmt"
	"sort"
	"strings"

	"github.com/cbrand/vodafone-billing-downloader/fetcher"
	"github.com/fatih/color"
	"github.com/rodaine/table"
)

const (
	USER_INFO_URL     = "https://api.vodafone.de/meinvodafone/v2/tmf-api/openid/v4/userinfo"
	USER_INFO_API_KEY = "aEIoMCae0A933wBL0bLlS6SwSBfkKwM5"
)

var (
	ErrUserInfoRequestFailed = errors.New("user info request failed")

	headerFmt = color.New(color.FgGreen, color.Underline).SprintfFunc()
	columnFmt = color.New(color.FgYellow).SprintfFunc()
)

func GetUserInfo(bearerToken fetcher.BearerToken) (*UserInfo, error) {
	var userInfo UserInfo
	headers := map[string]string{
		"x-api-key": USER_INFO_API_KEY,
	}
	err := fetcher.GetJsonWithHeaders(USER_INFO_URL, bearerToken, headers, &userInfo)
	if err == fetcher.ErrJsonRequestFailed {
		return nil, ErrUserInfoRequestFailed
	}
	return &userInfo, err
}

type UserInfo []UserInfoEntry

func (userInfo *UserInfo) HumanReadableString() string {
	if userInfo == nil || len(*userInfo) == 0 {
		return "=== User Account ===\n<empty>"
	}

	sections := []string{}
	for i, entry := range *userInfo {
		header := "=== User Account ==="
		if i > 0 {
			header = fmt.Sprintf("=== User Account (%d) ===", i+1)
		}
		sections = append(sections, header)
		sections = append(sections, entry.HumanReadableString())
	}

	return strings.Join(sections, "\n")
}

func (userInfo *UserInfo) GetActiveContractCableID() string {
	contractIDs := userInfo.GetAllContractIDs()
	if len(contractIDs) == 0 {
		return ""
	}
	return contractIDs[0]
}

func (userInfo *UserInfo) GetAllContractIDs() []string {
	if userInfo == nil {
		return nil
	}

	ids := map[string]struct{}{}
	for _, entry := range *userInfo {
		for _, asset := range entry.UserAssets {
			for _, id := range asset.ContractIDs() {
				if id == "" {
					continue
				}
				ids[id] = struct{}{}
			}
		}
	}

	contractIDs := make([]string, 0, len(ids))
	for id := range ids {
		contractIDs = append(contractIDs, id)
	}
	sort.Strings(contractIDs)
	return contractIDs
}

type UserInfoEntry struct {
	Title             string           `json:"title"`
	LevelOfAssurance  string           `json:"levelOfAssurance"`
	LastLoginDate     string           `json:"lastLoginDate"`
	IsPreferredEmail  bool             `json:"isPreferredUsernameEmailAddress"`
	Sub               string           `json:"sub"`
	Name              string           `json:"name"`
	Email             string           `json:"email"`
	EmailVerified     bool             `json:"email_verified"`
	GivenName         string           `json:"given_name"`
	FamilyName        string           `json:"family_name"`
	PreferredUsername string           `json:"preferred_username"`
	UserAssets        []*UserAsset     `json:"userAssets"`
	ExternalID        []*ExternalID    `json:"externalIdentifier"`
	Credentials       *UserCredentials `json:"credentials"`
	PhoneNumber       string           `json:"phoneNumber"`
	PhoneVerified     bool             `json:"phone_number_verified"`
}

func (entry *UserInfoEntry) HumanReadableString() string {
	profile := table.New("Field", "Value")
	profile.WithHeaderFormatter(headerFmt).WithFirstColumnFormatter(columnFmt)
	profile.AddRow("Name", entry.Name)
	profile.AddRow("Email", entry.Email)
	profile.AddRow("Preferred Username", entry.PreferredUsername)
	profile.AddRow("Last Login", entry.LastLoginDate)
	profile.AddRow("Level of Assurance", entry.LevelOfAssurance)

	profileData := bytes.NewBufferString("")
	profile.WithWriter(profileData).Print()

	assets := table.New("Contract ID", "Name", "Status", "Type")
	assets.WithHeaderFormatter(headerFmt).WithFirstColumnFormatter(columnFmt)
	for _, asset := range entry.UserAssets {
		contractIDs := asset.ContractIDs()
		if len(contractIDs) == 0 {
			continue
		}
		for _, contractID := range contractIDs {
			assets.AddRow(contractID, asset.Name, asset.Status, asset.AssetType)
		}
	}

	assetsData := bytes.NewBufferString("")
	assets.WithWriter(assetsData).Print()

	sections := []string{
		profileData.String(),
		"=== Assets ===",
		assetsData.String(),
	}
	return strings.Join(sections, "\n")
}

type UserCredentials struct {
	ID    string `json:"id"`
	State string `json:"state"`
}

type ExternalID struct {
	ID    string `json:"id"`
	Owner string `json:"owner"`
	Type  string `json:"type"`
}

type UserAsset struct {
	Name               string            `json:"name"`
	Status             string            `json:"status"`
	ID                 string            `json:"id"`
	AssetType          string            `json:"assetType"`
	EntityType         string            `json:"entityType"`
	Role               string            `json:"role"`
	ExternalIdentifier []*ExternalID     `json:"externalIdentifier"`
	RelatedAsset       []*RelatedAsset   `json:"relatedAsset"`
	Characteristic     []*Characteristic `json:"characteristic"`
}

func (asset *UserAsset) ContractIDs() []string {
	ids := []string{}
	for _, external := range asset.ExternalIdentifier {
		if external.Type == "customerNumber" {
			ids = append(ids, external.ID)
		}
	}
	if len(ids) > 0 {
		return ids
	}

	for _, related := range asset.RelatedAsset {
		for _, external := range related.ExternalIdentifier {
			if external.Type == "accountNumber" {
				ids = append(ids, external.ID)
			}
		}
	}

	return ids
}

type RelatedAsset struct {
	Type               string            `json:"@type"`
	Status             string            `json:"status"`
	Name               string            `json:"name"`
	ID                 string            `json:"id"`
	AssetType          string            `json:"assetType"`
	EntityType         string            `json:"entityType"`
	ExternalIdentifier []*ExternalID     `json:"externalIdentifier"`
	Characteristic     []*Characteristic `json:"characteristic"`
}

type Characteristic struct {
	Name  string `json:"name"`
	Value string `json:"value"`
}
