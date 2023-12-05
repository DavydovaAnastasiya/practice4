package main

import (
	"bufio"
	"fmt"
	"io"
	"log"
	"math/rand"
	"net"
	"net/http"
	"os"
	"os/exec"
	"strings"
	"sync"
	"time"
)

const TABLE_SIZE int = 512

var (
	mutex sync.Mutex
)
var requestCount int = 0

var requestTable = &RequestTable{}

func generateIp() string {
	var ipList [3]string
	ipList[0] = "192.168.1.1"
	ipList[1] = "192.168.1.2"
	ipList[2] = "192.168.1.3"
	return ipList[rand.Intn(len(ipList))]
}

type Request struct {
	url          string
	clientIp     string
	requestTime  time.Time
	timeInterval string
}

type RequestTable struct {
	requests [TABLE_SIZE]*Request
}

type JsonReport struct {
	id           int
	pid          string
	url          string
	sourceIP     string
	timeInterval string
	count        int
}

type HighItem struct {
	HighMap map[string]MiddleItem
	Count   int
}

type MiddleItem struct {
	MiddleMap map[string]LowItem
	Count     int
}

type LowItem struct {
	Count int
}

func GetColumnValue(name string, req Request) string {
	switch name {
	case "URL":
		return req.url
	case "SourceIP":
		return req.clientIp
	case "TimeInterval":
		return req.timeInterval
	}
	return ""
}

func generateStat(FirstColumn string, SecondColumn string, ThirdColumn string) map[string]HighItem {
	data := map[string]HighItem{}
	for _, item := range requestTable.requests {
		if item == nil {
			continue
		}
		firstValue, secondValue, thirdValue := GetColumnValue(FirstColumn, *item), GetColumnValue(SecondColumn, *item), GetColumnValue(ThirdColumn, *item)

		val, ok := data[firstValue]
		if !ok { //нет такого URL
			lowItem := &LowItem{Count: 1}
			middleItem := &MiddleItem{MiddleMap: map[string]LowItem{thirdValue: *lowItem}, Count: 1}
			highItem := &HighItem{HighMap: map[string]MiddleItem{secondValue: *middleItem}, Count: 1}
			data[firstValue] = *highItem
		} else { //есть такой URL
			val2, ok2 := val.HighMap[secondValue]
			if !ok2 { //нет такого IP
				lowItem := &LowItem{Count: 1}
				middleItem := &MiddleItem{MiddleMap: map[string]LowItem{thirdValue: *lowItem}, Count: 1}
				val2 = *middleItem
			} else { //есть такой IP
				val3, ok3 := val2.MiddleMap[thirdValue]
				if !ok3 { // нет такого timeInterval
					val3 = LowItem{Count: 1}
				} else { //есть такой timeInterval
					val3.Count++
				}
				val2.MiddleMap[thirdValue] = val3
				val2.Count++
			}
			val.HighMap[secondValue] = val2
			val.Count++
			data[firstValue] = val
		}
	}
	return data
}

func createReport(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost { //  проверка на метод запроса. Если метод не POST, то возвращается ошибка "Недопустимый метод"
		http.Error(w, "Недопустимый метод", http.StatusMethodNotAllowed)
		return
	}
	order, err := io.ReadAll(r.Body) //используется для чтения данных из тела запроса (потока r.Body), которые были отправлены в виде байтов, и сохранения их в переменную url.

	conn, err := net.Dial("tcp", "localhost:6411")
	if err != nil {
		fmt.Println("Не удалось подключиться к серверу:", err)
		os.Exit(1)
	}
	defer conn.Close() // отложенное закрытие соединения
	order2 := string(order)

	order2 = strings.Trim(order2, "[")
	order2 = strings.Replace(order2, "]", "", -1)
	order2 = strings.Replace(order2, "\"", "", -1)
	order2 = strings.Trim(order2, "\n")
	// a := strings.Split(order2, ", ")

	// FirstColumn := a[0]
	// SecondColumn := a[1]
	// ThirdColumn := a[2]
	FirstColumn := "SourceIP"
	SecondColumn := "TimeInterval"
	ThirdColumn := "URL"

	fmt.Println(FirstColumn)
	fmt.Println(SecondColumn)
	fmt.Println(ThirdColumn)

	data := generateStat(FirstColumn, SecondColumn, ThirdColumn)

	// log.Printf("%s\t|%s|\t%s|\tCount", FirstColumn, SecondColumn, ThirdColumn)
	response := ""
	for key, value := range data {
		response += fmt.Sprintf("%s\t%d\n", key, value.Count)
		for key2, value2 := range value.HighMap {
			response += fmt.Sprintf("\t%s\t%d\n", key2, value2.Count)
			for key3, value3 := range value2.MiddleMap {
				response += fmt.Sprintf("\t\t%s\t%d\n", key3, value3.Count)
			}
		}
	}
	fmt.Fprintf(w, "Детализация: \n%s", response) // отправка короткой ссылки клиенту
}

func createJSON(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost { //  проверка на метод запроса. Если метод не POST, то возвращается ошибка "Недопустимый метод"
		http.Error(w, "Недопустимый метод", http.StatusMethodNotAllowed)
		return
	}

	conn, err := net.Dial("tcp", "localhost:6411")
	if err != nil {
		fmt.Println("Не удалось подключиться к серверу:", err)
		os.Exit(1)
	}
	defer conn.Close() // отложенное закрытие соединения

	FirstColumn := "SourceIP"
	SecondColumn := "TimeInterval"
	ThirdColumn := "URL"
	data := generateStat(FirstColumn, SecondColumn, ThirdColumn)
	countJson := 0

	response := "[\n"

	for key, value := range data {
		response += fmt.Sprintf("\t{\n\t\t\"ID\":%d\n\t\t\"PID\":%s\n\t\t\"URL\":%s\n\t\t\"SourceIP\":%s\n\t\t\"TimeInterval\":%s\n\t\t\"PID\":%d\n\t},\n", countJson+1, "null", key, "null", "null", value.Count)
		countJson++
		pidValue := countJson
		for key2, value2 := range value.HighMap {
			response += fmt.Sprintf("\t{\n\t\t\"ID\":%d\n\t\t\"PID\":%d\n\t\t\"URL\":%s\n\t\t\"SourceIP\":%s\n\t\t\"TimeInterval\":%s\n\t\t\"PID\":%d\n\t},\n", countJson+1, pidValue, "null", key2, "null", value2.Count)
			countJson++
			pidValue2 := countJson
			for key3, value3 := range value2.MiddleMap {
				response += fmt.Sprintf("\t{\n\t\t\"ID\":%d\n\t\t\"PID\":%d\n\t\t\"URL\":%s\n\t\t\"SourceIP\":%s\n\t\t\"TimeInterval\":%s\n\t\t\"PID\":%d\n\t},\n", countJson+1, pidValue2, "null", "null", key3, value3.Count)
				countJson++
			}
		}
	}
	response += "]	"
	fmt.Fprintf(w, "%s", response) // отправка короткой ссылки клиенту
}

// shortenHandler обрабатывает POST запросы на создание сокращенной ссылки.
//принимает URL-адрес из тела запроса, генерирует короткую ссылку и сохраняет соответствие между короткой и оригинальной ссылками в базе данных.

func shortenHandler(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost { //  проверка на метод запроса. Если метод не POST, то возвращается ошибка "Недопустимый метод"
		http.Error(w, "Недопустимый метод", http.StatusMethodNotAllowed)
		return
	}
	url, err := io.ReadAll(r.Body) //используется для чтения данных из тела запроса (потока r.Body), которые были отправлены в виде байтов, и сохранения их в переменную url.

	conn, err := net.Dial("tcp", "localhost:6411")
	if err != nil {
		fmt.Println("Не удалось подключиться к серверу:", err)
		os.Exit(1)
	}
	defer conn.Close() // отложенное закрытие соединения

	mutex.Lock()
	defer mutex.Unlock()

	shortURL := generateShortURL()

	fmt.Printf("URL: [%s]\t Short: [%s]\n", string(url), shortURL)

	// генерация короткой ссылки
	_, err = conn.Write([]byte("HSET " + shortURL + " " + string(url) + "\n")) // запись в БД оригинальной и короткой ссылок
	if err != nil {
		fmt.Println("Не удалось отправить команду на сервер:", err)
		return
	}

	resp, err := bufio.NewReader(conn).ReadString('\n')

	log.Printf("resp: [%s]\n", resp)

	if strings.Contains(resp, "This link is already in our base") {
		log.Println("This link is already in our base!")
		fmt.Fprintf(w, resp)
		return
	}

	log.Println("Passed!")
	fmt.Fprintf(w, "Сокращенная ссылка: http://localhost:8010/%s", shortURL) // отправка короткой ссылки клиенту
}

// redirectHandler обрабатывает GET запросы для перенаправления на оригинальную ссылку.
// получает короткую ссылку из URL запроса, ищет соответствующую оригинальную ссылку в базе данных и перенаправляет клиента на нее.
func redirectHandler(w http.ResponseWriter, r *http.Request) {
	conn, err := net.Dial("tcp", "localhost:6411")
	if err != nil {
		fmt.Println("Не удалось подключиться к серверу:", err)
		os.Exit(1)
	}
	defer conn.Close()

	mutex.Lock()
	defer mutex.Unlock()

	shortURL := strings.TrimPrefix(r.URL.Path, "/") // получение короткой ссылки из URL запроса

	// for _, item := range requestTable.requests {
	// 	if item == nil {
	// 		continue
	// 	}
	// 	log.Printf("URL: %s, IP: %s, TIME: %s\n", item.url, item.clientIp, item.requestTime)
	// }

	if shortURL == "" || shortURL == "favicon.ico" {
		conn.Write([]byte("Тут ничего нет..."))
		return
	}

	fmt.Printf("URL: [%s]\t Short: [%s]\n", string(r.URL.Path), shortURL)

	_, err = conn.Write([]byte("HGET " + shortURL + "\n")) // чтение из БД оригинальной ссылки по короткой ссылке
	if err != nil {
		fmt.Println("Не удалось отправить команду на сервер:", err)
		http.NotFound(w, r) // если не удалось отправить команду, возвращаем ошибку 404
		return
	}

	log.Println("Died here!")

	resp, err := bufio.NewReader(conn).ReadString('\n') // чтение ответа от сервера

	log.Println(resp)

	if err != nil {
		fmt.Fprintf(w, "Server Error!")
		//http.NotFound(w, r) // если не удалось прочитать ответ, возвращаем ошибку 404
		return
	}

	log.Printf("resp: [%s]\n", resp)

	if resp == "Эта ссылка уже есть!\n" {
		log.Println("This link is already in our base!")
		fmt.Fprintf(w, "This link is already in our base!")
		return
	}

	originalURL := resp // получение оригинальной ссылки из ответа сервера

	timeNow := time.Now()
	hourRequest := timeNow.Hour()
	minuteRequest := timeNow.Minute()
	strData := ""
	if minuteRequest < 59 {
		strData = fmt.Sprintf("%d:%d-%d:%d", hourRequest, minuteRequest, hourRequest, minuteRequest+1)
	} else {
		if hourRequest == 23 && minuteRequest == 59 {
			strData = fmt.Sprintf("%d:%d-%d:%d", hourRequest, minuteRequest, 0, 0)
		} else {
			strData = fmt.Sprintf("%d:%d-%d:%d", hourRequest, minuteRequest, hourRequest+1, 0)
		}
	}

	request := &Request{url: strings.Trim(originalURL, "\n") + "(" + shortURL + ")", clientIp: generateIp(), requestTime: timeNow, timeInterval: strData}
	requestTable.requests[requestCount] = request
	requestCount++

	http.Redirect(w, r, "/", http.StatusFound)                // перенаправление клиента на оригинальную ссылку
	exec.Command("cmd.exe", "/C", "start", originalURL).Run() //открыть оригинальный URL в стандартном веб-браузере операционной системы
}

// generateShortURL генерирует случайную короткую ссылку
func generateShortURL() string {
	const charset = "abcdefghijklmnopqrstuvwxyzABCDEFGHIJKLMNOPQRSTUVWXYZ0123456789" // набор символов для генерации короткой ссылки
	b := make([]byte, 7)                                                             // создание байтового массива размером 7 байт
	for i := range b {                                                               // заполнение массива случайными символами из набора charset
		b[i] = charset[rand.Intn(len(charset))]
	}
	return string(b) // возвращение сгенерированной короткой ссылки в виде строки
}

func main() {

	http.HandleFunc("/shorten", shortenHandler) // обработчик POST запросов на создание сокращенной ссылки
	//вызывается, когда клиент отправляет POST запрос на адрес "/shorten"
	http.HandleFunc("/", redirectHandler) // обработчик GET запросов для перенаправления на оригинальную ссылку
	// вызывается, когда клиент отправляет GET запрос на любой другой адрес, кроме "/shorten".
	http.HandleFunc("/detailed_stats", createReport) // обработчик POST запросов на создание сокращенной ссылки
	//вызывается, когда клиент отправляет POST запрос на адрес "/detailed_stats"
	http.HandleFunc("/json_stats", createJSON) // обработчик POST запросов на создание сокращенной ссылки
	//вызывается, когда клиент отправляет POST запрос на адрес "/json_stats"

	fmt.Println("Сервер запущен на порту 8010")
	err := http.ListenAndServe(":8010", nil) // запуск сервера на порту
	if err != nil {
		fmt.Println("Ошибка запуска сервера:", err)
	}
}
