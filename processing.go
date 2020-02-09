package main

import "fmt"

func index(repoPath string, packets []string) error {
	if len(packets) == 0 {
		fmt.Println("обработка всех пакетов")
	} else {
		for _, pack := range packets {
			fmt.Printf("обработка пакета: %v\n", pack)
		}
	}
}
