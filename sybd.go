package main

import (
	"bufio"
	"errors"
	"fmt"
	"log"
	"net"
	"strings"
	"sync"
)

const TABLE_SIZE int = 512

type Element struct {
	key   string
	value string
}

type HashMap struct {
	hashmap [TABLE_SIZE]*Element //поле hashmap - массив указателей на структуру Element
	mutex   sync.Mutex
}

func HashFunc(key string) int {
	sum := 0
	for _, char := range key { //проходимся по каждой букве ключа
		sum += int(char) //преобразуем букву в число
	}
	return sum % TABLE_SIZE //возвращаем хеш значение
}

// вставка нового элемента в хеш-таблицу
func (hmap *HashMap) insert(key string, value string) error {
	Element := &Element{key: key, value: value} //создается новый элемент типа Element с переданными ключем и значением
	index := HashFunc(key)
	if hmap.hashmap[index] == nil {
		hmap.hashmap[index] = Element
		return nil
	} else {
		if hmap.hashmap[index].key == key {
			hmap.hashmap[index] = Element //элемент заменяется новым
			return nil
		}
		index++
		for i := 0; i < TABLE_SIZE; i++ { //проходимся по таблице до конца
			if index == TABLE_SIZE {
				index = 0 //если дошли до конца, начинаем проверку с начала
			}
			if hmap.hashmap[index] == nil {
				hmap.hashmap[index] = Element //поиск ближайшего свободного индекса для вставки элемента
				return nil
			}
			index++
		}
	}
	fmt.Println("Недостаточно пространства!") //если все индексы хеш-таблицы заняты, выводится сообщение об ошибке и возвращается соответствующая ошибка
	return errors.New("недостаточно пространства")
}

// удаление элемента из хеш-таблицы
func (hmap *HashMap) delete(key string) error {
	index := HashFunc(key)

	if hmap.hashmap[index] == nil {
		fmt.Println("Объект не найден!")
		return errors.New("объект не найден")
	}

	if hmap.hashmap[index] == nil || hmap.hashmap[index].key != key {
		return errors.New("Элемент не найден")
	}

	if hmap.hashmap[index].key == key {
		hmap.hashmap[index] = nil //элемент удаляется путем присвоения значения nil
		return nil
	} else {
		index++
		for index < TABLE_SIZE {
			if hmap.hashmap[index].key == key { //поиск элемента с данным ключом на следующих индексах, если элемент найден - он удаляется
				hmap.hashmap[index] = nil
				return nil
			}
			index++
		}
	}
	return errors.New("недостаточно пространства")
}

// получение элемента из хеш-таблицы по ключу
func (hmap *HashMap) get(key string) (string, error) {
	index := HashFunc(key)
	if hmap.hashmap[index] == nil {
		fmt.Println("Объекта с таким ключом нет в хеш-таблице!")
		return "", errors.New("объект не найден")
	}

	if hmap.hashmap[index].key == key { //проверка совпадения ключей
		fmt.Printf("Объект найден! ключ: [%s] значение: [%s]\n", hmap.hashmap[index].key, hmap.hashmap[index].value)
		return hmap.hashmap[index].value, nil
	} else {
		index++
		for i := 0; i < TABLE_SIZE; i++ { //проходимся по таблице до конца
			if index == TABLE_SIZE {
				index = 0 //если дошли до конца, начинаем проверку с начала
			}

			if hmap.hashmap[index] == nil {
				return "", errors.New("объект не найден")
			}

			if hmap.hashmap[index].key == key {
				received_value := hmap.hashmap[index].value
				fmt.Printf("Объект найден! ключ: [%s] значение: [%s]\n", hmap.hashmap[index].key, received_value)
				return received_value, nil
			}
			index++
		}
	}
	fmt.Printf("Объект не найден!")
	return "", errors.New("объект не найден")
}

func parser(conn net.Conn, command string, hashmap *HashMap, linkShort *HashMap) {
	commandParts := strings.Fields(command)

	fmt.Println(commandParts)

	if len(commandParts) < 2 { // проверка наличия двух аргументов в команде
		fmt.Fprintln(conn, "Недостаточно аргументов")
		return
	}

	key := commandParts[1]
	switch commandParts[0] {
	case "HSET":
		if len(commandParts) != 3 { // проверка наличия трех аргументов в команде HSET
			fmt.Fprintln(conn, "Недостаточно аргументов")
			return
		}
		value := commandParts[2]

		hashmap.mutex.Lock()
		linkShort.mutex.Lock()
		defer hashmap.mutex.Unlock()
		defer linkShort.mutex.Unlock()

		short, err := linkShort.get(value)

		if err == nil {
			log.Println("Эта ссылка уже есть!")
			conn.Write([]byte("This link is already in our base: " + short + "\n"))
			return
		}

		if err := hashmap.insert(key, value); err != nil { // вставка элемента в хеш-таблицу с указанным ключом и значением
			fmt.Fprintln(conn, "Ошибка вставки:", err)
			return
		}

		linkShort.insert(value, key)

		// вывод всех элементов из обеих таблиц на экран для проверки корректности работы программы
		for _, item := range hashmap.hashmap {
			if item == nil {
				continue
			}
			log.Printf("KEY: %s, VALUE: %s\n", item.key, item.value)
		}

		for _, item := range linkShort.hashmap {
			if item == nil {
				continue
			}
			log.Printf("VALUE: %s, KEY: %s\n", item.key, item.value)
		}

		fmt.Fprintln(conn, value)
		return
	case "HGET":
		hashmap.mutex.Lock()
		defer hashmap.mutex.Unlock()

		for _, item := range hashmap.hashmap {
			if item == nil {
				continue
			}
			log.Printf("KEY: %s, VALUE: %s\n", item.key, item.value)
		}

		if value, err := hashmap.get(key); err != nil { // получение значения из хеш-таблицы по заданному ключу
			log.Println("Ошибка получения значения:", err.Error())
			conn.Write([]byte("No link found!\n"))
			return
		} else {
			fmt.Fprintln(conn, value)
		}

		return
	default:
		fmt.Fprintln(conn, "Недопустимая команда") //вывод сообщения об ошибке, если команда не соответствует "HSET" или "HGET"
		return
	}
}

func main() {
	hashMap := &HashMap{}
	linkShort := &HashMap{}

	listener, err := net.Listen("tcp", ":6411") // метод net.Listen используется для создания слушателя (listener), который принимает входящие соединения

	if err != nil {
		fmt.Println("Ошибка при запуске сервера:", err)
		return
	}
	defer listener.Close()
	fmt.Println("Прослушивание...")
	for {
		conn, err := listener.Accept() // используя метод listener.Accept(), сервер ожидает подключения клиента
		if err != nil {
			fmt.Println("Ошибка при принятии соединения:", err)
			conn.Close()
			continue
		}
		go handleConnection(conn, hashMap, linkShort)
	}
}

func handleConnection(conn net.Conn, hashMap *HashMap, linkShort *HashMap) { //функция handleConnection отвечает за обработку входящего соединения conn
	defer conn.Close() // ключевое слово defer гарантирует, что соединение будет закрыто после завершения функции
	source := bufio.NewScanner(conn)

	for source.Scan() {
		command := source.Text()
		parser(conn, command, hashMap, linkShort)
	}
}
