package main

import (
	"fmt"
	"log"
	"github.com/fanatcx/gator/internal/config"
)

func main() {
	
	jsonObject, err := config.Read()
	if err != nil {
		log.Fatal(err)
	}

	jsonObject.SetUser("Chris")
	fmt.Printf("Current username: %s\n", jsonObject.CurrentUserName)
	fmt.Print("Current object on disk: ")
	fmt.Println(jsonObject)


}