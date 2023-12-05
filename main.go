package main

import (
	"bufio"
	"bytes"
	"fmt"
	"io"
	"log"
	"net/http"
	"os"
	"strings"
)

func main() {
	scanner := bufio.NewScanner(os.Stdin)

	for {

		fmt.Println("1. Сокращение ссылки")
		fmt.Println("2. Переход по сокращенной ссылке")
		fmt.Println("3. Запросить детализированный отчет")
		fmt.Println("4. Запросить json отчет")
		fmt.Println("5. Выход")
		fmt.Print("Введите номер команды: ")

		scanner.Scan()
		input := scanner.Text()
		switch input {
		case "1":
			{
				fmt.Println("Введите ссылку: ")
				link, _ := bufio.NewReader(os.Stdin).ReadString('\n')
				link = link[:len(link)-2] // удаляются символы переноса строки

				fmt.Printf("Link: [%s]\n", link) // выводится введенная ссылка в квадратных скобках

				req, err := http.NewRequest("POST", "http://localhost:8010/shorten", bytes.NewBuffer([]byte(link))) // создается POST-запрос на сервер
				if err != nil {
					fmt.Println(err.Error())
					continue
				}

				res, err := http.DefaultClient.Do(req) // выполняется запрос на сервер

				if err != nil {
					fmt.Println("Ошибка при отправке запроса!")
					continue
				}
				defer res.Body.Close()

				body, _ := io.ReadAll(res.Body) // читается тело ответа сервера

				fmt.Println(string(body)) // выводится сокращенная ссылка, полученная от сервера
				break
			}
		case "2":
			fmt.Println("Введите сокращение: ")                   // выводится приглашение ввести сокращенную ссылку
			link, _ := bufio.NewReader(os.Stdin).ReadString('\n') // считывается введенная строка с консоли
			link = strings.Trim(link, "\n")                       // удаляются символы переноса строки
			link = fmt.Sprintf("http://localhost:8010/%s", link)  // формируется адрес сервера для перехода по сокращенной ссылке
			link = strings.TrimRight(link, "\r")

			res, err := http.Get(link) // выполняется GET-запрос на сервер

			if err != nil {
				log.Fatal(err)
			}

			answer, err := io.ReadAll(res.Body) // читается тело ответа сервера

			if err != nil {
				continue
			}

			fmt.Println(string(answer)) // выводится оригинальная ссылка, полученная от сервера

			defer res.Body.Close()
			break

		case "3":
			fmt.Println("Введите порядок детализации например: [\"SourceIP\", \"TimeInterval\", \"URL\"]: ")
			order, _ := bufio.NewReader(os.Stdin).ReadString('\n')
			req, err := http.NewRequest("POST", "http://localhost:8010/detailed_stats", bytes.NewBuffer([]byte(order))) // создается POST-запрос на сервер
			if err != nil {
				fmt.Println(err.Error())
				continue
			}
			res, err := http.DefaultClient.Do(req) // выполняется запрос на сервер
			defer res.Body.Close()

			body, _ := io.ReadAll(res.Body) // читается тело ответа сервера

			fmt.Println(string(body)) // выводится сокращенная ссылка, полученная от сервера
			break

		case "4":
			order := ""
			req, err := http.NewRequest("POST", "http://localhost:8010/json_stats", bytes.NewBuffer([]byte(order)))
			// создается POST-запрос на сервер, хотя в данном коде пустое тело запроса отправляется,
			//но сервер может использовать этот запрос для выполнения определенных действий или генерации JSON-отчета на основе
			//различных параметров или данных, которые могут быть переданы в теле запроса или через другие методы
			if err != nil {
				fmt.Println(err.Error())
				continue
			}
			res, err := http.DefaultClient.Do(req) // выполняется запрос на сервер
			defer res.Body.Close()

			body, _ := io.ReadAll(res.Body) // читается тело ответа сервера

			fmt.Println(string(body))
			break

		case "5":
			return
		default:
			fmt.Println("Неправильный ввод!")
		}

	}

}
