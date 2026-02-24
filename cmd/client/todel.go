package main

import (
	"bufio"
	"fmt"
	"log"
	"net/http"
	"net/url"
	"os"
	"strings"
)

func toDel() bool {
	endpoint := "http://localhost:8080/"
	// контейнер данных для запроса
	data := url.Values{}
	// приглашение в консоли
	fmt.Println("Введите длинный URL")
	// открываем потоковое чтение из консоли
	reader := bufio.NewReader(os.Stdin)
	// читаем строку из консоли
	long, err := reader.ReadString('\n')
	if err != nil {
		log.Fatalln(err)
	}

	long = strings.TrimSuffix(long, "\n")
	log.Println("long", long)
	request, err := http.NewRequest(http.MethodDelete, endpoint, strings.NewReader(data.Encode()))
	if err != nil {
		log.Fatalln(err)
	}
	// в заголовках запроса указываем кодировку
	request.Header.Add("Content-Type", "application/x-www-form-urlencoded")

	return false
}
