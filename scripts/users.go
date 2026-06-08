package main

import (
    _ "embed"
    "bytes"
    "encoding/json"
    "fmt"
    "net/http"
	"time"

    "github.com/brianvoe/gofakeit/v7"
)

//go:embed fixtures/users.json
var usersFixture []byte

type registerPayload struct {
    Username string `json:"username"`
    Email    string `json:"email"`
    Password string `json:"password"`
    BirthDate string `json:"birthDate"`
    Gender    string `json:"gender"`
}
type authResponse struct {
    Token string `json:"token"`
}

func seedUsers(apiURL string, generatedCount int) ([]string, error) {
    // 1. Charge les users fixes depuis le JSON
    var fixed []registerPayload
    if err := json.Unmarshal(usersFixture, &fixed); err != nil {
        return nil, fmt.Errorf("parse fixtures/users.json: %w", err)
    }

    // 2. Génère des users supplémentaires
    generated := make([]registerPayload, generatedCount)
    for i := range generated {
    generated[i] = registerPayload{
        Username:  gofakeit.Username(),
        Email:     gofakeit.Email(),
        Password:  "password123",
        BirthDate: gofakeit.DateRange(
            time.Date(1970, 1, 1, 0, 0, 0, 0, time.UTC),
            time.Date(2005, 1, 1, 0, 0, 0, 0, time.UTC),
        ).Format("2006-01-02"),
        Gender: gofakeit.RandomString([]string{"male", "female", "other"}),
    }
    }

    // 3. Appelle l'API pour tous
    var tokens []string
    all := append(fixed, generated...)

    for _, u := range all {
        token, err := registerUser(apiURL, u)
        if err != nil {
            fmt.Printf("  ⚠️  %s : %v\n", u.Email, err)
            continue
        }
        tokens = append(tokens, token)
    }

    fmt.Printf("  ✅ %d utilisateurs créés (%d fixes + %d générés)\n",
        len(tokens), len(fixed), generatedCount)

    return tokens, nil
}

func registerUser(apiURL string, payload registerPayload) (string, error) {
    body, _ := json.Marshal(payload)

    resp, err := http.Post(
        apiURL+"/auth/register",
        "application/json",
        bytes.NewBuffer(body),
    )
    if err != nil {
        return "", err
    }
    defer resp.Body.Close()

    if resp.StatusCode != http.StatusCreated && resp.StatusCode != http.StatusOK {
        return "", fmt.Errorf("status %d", resp.StatusCode)
    }

    var result authResponse
    json.NewDecoder(resp.Body).Decode(&result)

    return result.Token, nil
}