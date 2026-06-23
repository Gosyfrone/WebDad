package database

import "testing"

func TestConnectMongo_InvalidURI(t *testing.T) {
	if _, err := ConnectMongo("://bad-uri"); err == nil {
		t.Fatal("URI Mongo invalide devrait retourner une erreur")
	}
}
