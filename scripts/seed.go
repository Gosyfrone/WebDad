package main

import (
    "fmt"
    "log"
    "os"
)

func main() {
    apiURL := os.Getenv("API_URL")
    if apiURL == "" {
        apiURL = "http://localhost:8080"
    }

    fmt.Println("🌱 Seeding via API Gateway...")
    fmt.Println()

    // 1. Users → récupère les tokens pour les appels suivants
    fmt.Println("👤 Création des utilisateurs...")
    _, err := seedUsers(apiURL, 20)
    if err != nil {
        log.Fatalf("❌ seedUsers: %v", err)
    }


    fmt.Println()
    fmt.Println("✅ Seed terminé !")
}