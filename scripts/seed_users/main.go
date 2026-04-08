package main

import (
	"fmt"
	"log"

	"github.com/bulgadev/pushUpCounter/utils"
)

type challengeUser struct {
	username string
	password string
}

func main() {
	db, err := utils.OpenDuckDB("data.db")
	if err != nil {
		log.Fatal(err)
	}
	defer db.Close()

	if err := utils.EnsureUsersTableSchema(db); err != nil {
		log.Fatal(err)
	}

	users := []challengeUser{
		{username: "neguebas", password: "PequenoNeguin166@"},
		{username: "rick", password: "castelaostuker67@"},
		{username: "marcelosvitolas", password: "VoltaPraMimAnaCarolina12@"},
	}

	for _, user := range users {
		if err := utils.UpsertChallengeUser(db, user.username, user.password); err != nil {
			log.Fatalf("failed to seed user %s: %v", user.username, err)
		}
		fmt.Printf("seeded user: %s\n", user.username)
	}

	fmt.Println("done")
}
