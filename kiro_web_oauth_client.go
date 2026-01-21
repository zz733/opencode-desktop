package main

import (
	"bytes"
	"fmt"
	"io"
	"net/http"
	"time"

	"github.com/fxamacker/cbor/v2"
)

const (
	KiroWebPortalURL = "https://app.kiro.dev"
)

type KiroWebPortalClient struct {
	client   *http.Client
	endpoint string
}

func NewKiroWebPortalClient() *KiroWebPortalClient {
	return &KiroWebPortalClient{
		client:   &http.Client{Timeout: 30 * time.Second},
		endpoint: KiroWebPortalURL,
	}
}

// --- Request/Response Structs ---

type InitiateLoginRequest struct {
	Idp                 string `cbor:"idp"`
	RedirectUri         string `cbor:"redirectUri"`
	CodeChallenge       string `cbor:"codeChallenge"`
	CodeChallengeMethod string `cbor:"codeChallengeMethod"`
	State               string `cbor:"state"`
}

type InitiateLoginResponse struct {
	RedirectUrl string `cbor:"redirectUrl"`
}

type ExchangeTokenRequest struct {
	Idp          string `cbor:"idp"`
	Code         string `cbor:"code"`
	CodeVerifier string `cbor:"codeVerifier"`
	RedirectUri  string `cbor:"redirectUri"`
	State        string `cbor:"state"`
}

type ExchangeTokenCborResponse struct {
	AccessToken string `cbor:"accessToken"`
	CsrfToken   string `cbor:"csrfToken"`
	ExpiresIn   int64  `cbor:"expiresIn"`
	ProfileArn  string `cbor:"profileArn"`
}

type ExchangeTokenResult struct {
	AccessToken  string
	CsrfToken    string
	ExpiresIn    int64
	ProfileArn   string
	SessionToken string // From Set-Cookie (RefreshToken)
	Idp          string // From Set-Cookie
}

type GetUserInfoResponse struct {
	Email        string      `cbor:"email"`
	UserId       string      `cbor:"userId"`
	Idp          string      `cbor:"idp"`
	Status       string      `cbor:"status"`
	FeatureFlags interface{} `cbor:"featureFlags"`
}

type GetUserUsageAndLimitsRequest struct {
	IsEmailRequired bool   `cbor:"isEmailRequired"`
	Origin          string `cbor:"origin"`
}

// Usage structs... (omitted for now, can add if needed for usage display)

// --- Methods ---

func (c *KiroWebPortalClient) InitiateLogin(idp, redirectUri, codeChallenge, state string) (*InitiateLoginResponse, error) {
	url := fmt.Sprintf("%s/service/KiroWebPortalService/operation/InitiateLogin", c.endpoint)

	reqBody := InitiateLoginRequest{
		Idp:                 idp,
		RedirectUri:         redirectUri,
		CodeChallenge:       codeChallenge,
		CodeChallengeMethod: "S256",
		State:               state,
	}

	data, err := cbor.Marshal(reqBody)
	if err != nil {
		return nil, fmt.Errorf("CBOR marshal error: %w", err)
	}

	req, err := http.NewRequest("POST", url, bytes.NewReader(data))
	if err != nil {
		return nil, err
	}
	req.Header.Set("Content-Type", "application/cbor")
	req.Header.Set("Accept", "application/cbor")
	req.Header.Set("smithy-protocol", "rpc-v2-cbor")

	resp, err := c.client.Do(req)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		body, _ := io.ReadAll(resp.Body)
		return nil, fmt.Errorf("InitiateLogin failed (%d): %s", resp.StatusCode, string(body))
	}

	var result InitiateLoginResponse
	if err := cbor.NewDecoder(resp.Body).Decode(&result); err != nil {
		return nil, fmt.Errorf("CBOR decode error: %w", err)
	}

	return &result, nil
}

func (c *KiroWebPortalClient) ExchangeToken(idp, code, codeVerifier, redirectUri, state string) (*ExchangeTokenResult, error) {
	url := fmt.Sprintf("%s/service/KiroWebPortalService/operation/ExchangeToken", c.endpoint)

	reqBody := ExchangeTokenRequest{
		Idp:          idp,
		Code:         code,
		CodeVerifier: codeVerifier,
		RedirectUri:  redirectUri,
		State:        state,
	}

	data, err := cbor.Marshal(reqBody)
	if err != nil {
		return nil, fmt.Errorf("CBOR marshal error: %w", err)
	}

	req, err := http.NewRequest("POST", url, bytes.NewReader(data))
	if err != nil {
		return nil, err
	}
	req.Header.Set("Content-Type", "application/cbor")
	req.Header.Set("Accept", "application/cbor")
	req.Header.Set("smithy-protocol", "rpc-v2-cbor")

	resp, err := c.client.Do(req)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		body, _ := io.ReadAll(resp.Body)
		return nil, fmt.Errorf("ExchangeToken failed (%d): %s", resp.StatusCode, string(body))
	}

	// Read body first to decode CBOR
	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, err
	}

	var cborResp ExchangeTokenCborResponse
	if err := cbor.Unmarshal(body, &cborResp); err != nil {
		return nil, fmt.Errorf("CBOR decode error: %w", err)
	}

	// Parse Cookies
	var sessionToken, cookieIdp string
	for _, cookie := range resp.Cookies() {
		switch cookie.Name {
		case "RefreshToken":
			sessionToken = cookie.Value
		case "Idp":
			cookieIdp = cookie.Value
		}
	}

	return &ExchangeTokenResult{
		AccessToken:  cborResp.AccessToken,
		CsrfToken:    cborResp.CsrfToken,
		ExpiresIn:    cborResp.ExpiresIn,
		ProfileArn:   cborResp.ProfileArn,
		SessionToken: sessionToken,
		Idp:          cookieIdp,
	}, nil
}

type GetUserInfoRequest struct {
	Origin string `cbor:"origin"`
}

func (c *KiroWebPortalClient) GetUserInfo(accessToken, csrfToken, refreshToken, idp string) (*GetUserInfoResponse, error) {
	url := fmt.Sprintf("%s/service/KiroWebPortalService/operation/GetUserInfo", c.endpoint)

	reqBody := GetUserInfoRequest{
		Origin: "KIRO_IDE",
	}

	data, err := cbor.Marshal(reqBody)
	if err != nil {
		return nil, fmt.Errorf("CBOR marshal error: %w", err)
	}

	req, err := http.NewRequest("POST", url, bytes.NewReader(data))
	if err != nil {
		return nil, err
	}
	req.Header.Set("Content-Type", "application/cbor")
	req.Header.Set("Accept", "application/cbor")
	req.Header.Set("smithy-protocol", "rpc-v2-cbor")
	req.Header.Set("Authorization", fmt.Sprintf("Bearer %s", accessToken))

	// Set Cookies
	cookie := fmt.Sprintf("Idp=%s; AccessToken=%s", idp, accessToken)
	req.Header.Set("Cookie", cookie)

	resp, err := c.client.Do(req)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		body, _ := io.ReadAll(resp.Body)
		return nil, fmt.Errorf("GetUserInfo failed (%d): %s", resp.StatusCode, string(body))
	}

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, err
	}

	var result GetUserInfoResponse
	if err := cbor.Unmarshal(body, &result); err != nil {
		return nil, fmt.Errorf("CBOR decode error: %w", err)
	}

	return &result, nil
}
