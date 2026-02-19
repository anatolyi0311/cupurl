package main

import (
	"bufio"
	"fmt"
	"io"
	"log"
	"net/http"
	"net/url"
	"os"
	"strings"
	"time"
)

func main() {
	easyPost()
}

func easyPost() {
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
	fmt.Println("long:", long)
	// заполняем контейнер данными
	data.Set("url", long)
	// добавляем HTTP-клиент
	client := &http.Client{
		Timeout: time.Duration(30 * time.Second),
	}
	// пишем запрос
	// запрос методом POST должен, помимо заголовков, содержать тело
	// тело должно быть источником потокового чтения io.Reader
	request, err := http.NewRequest(http.MethodPost, endpoint, strings.NewReader(data.Get("url"))) // data.Encode()
	if err != nil {
		log.Fatalln(err)
	}
	// в заголовках запроса указываем кодировку
	// request.Header.Add("Content-Type", "application/x-www-form-urlencoded")
	request.Header.Add("Content-Type", "text/plain")
	// отправляем запрос и получаем ответ
	response, err := client.Do(request)
	if err != nil {
		log.Fatalln(err)
	}
	// выводим код ответа
	fmt.Println("Статус-код ", response.Status)
	defer response.Body.Close()
	// читаем поток из тела ответа
	body, err := io.ReadAll(response.Body)
	if err != nil {
		log.Fatalln(err)
	}
	// и печатаем его
	fmt.Println("Response ", string(body))

	// GET
	parsedLink, err := url.Parse(string(body))
	if err != nil {
		log.Fatalln("Fatal.Parse:", err)
	}
	fmt.Println("parsedLink.Path ", parsedLink.Path)
	request2, err := http.NewRequest(http.MethodGet, string(body), nil) // data.Encode()
	if err != nil {
		log.Fatalln("Fatal.req:", err)
	}
	response2, err := client.Do(request2)
	if err != nil {
		log.Fatalln("Fatal.res:", err)
	}
	fmt.Println("Статус-код ", response2.Status)
	defer response2.Body.Close()
	fmt.Println("Response ", response2.Header)

}
