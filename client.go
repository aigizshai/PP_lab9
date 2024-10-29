package main

import (
	"bytes"
	"encoding/json"
	"fmt"
	"net/http"
	"os"
)

const baseURL = "http://localhost:8080/users"

type User struct {
	ID   string `json:"id,omitempty"`
	Name string `json:"name`
	Age  string `json:"age`
}

func main() {
	for {
		fmt.Println("\nВыберите действие:")
		fmt.Println("1. Показать всех пользователей")
		fmt.Println("2. Найти пользователся по имени")
		fmt.Println("3. Добавить нового пользователя")
		fmt.Println("4. Обновить данные пользователя")
		fmt.Println("5. Удалить пользователя")
		fmt.Println("0. Выход")

		var choice int
		fmt.Scan(&choice)

		switch choice {
		case 1:
			getUsers()
		case 2:
			findUserByName()
		case 3:
			createUser()
		case 4:
			updateUser()
		case 5:
			deleteUser()
		case 0:
			os.Exit(0)
		default:
			fmt.Println("Неккоректный ввод")
		}

	}
}

func hadleResponse(resp *http.Response, successMsg string) {
	defer resp.Body.Close()
	if resp.StatusCode >= 200 && resp.StatusCode < 300 {
		fmt.Println(successMsg)
	} else {
		fmt.Println("Ошибка: ", resp.Body)
	}
}
func handleError(msg string, err error) {
	fmt.Printf("%s %s\n", msg, err)
}

func printUsers(users []User) {
	if len(users) == 0 {
		fmt.Println("Нет данных")
		return
	}
	fmt.Println("\nСписок пользователей")
	for _, user := range users {
		fmt.Printf("ID: %s, Имя: %s, Возраст: %s\n", user.ID, user.Name, user.Age)
	}
}

func getUsers() {
	resp, err := http.Get(baseURL)
	if err != nil {
		handleError("Ошибка при получении пользователей", err)
	}
	defer resp.Body.Close()

	var users []User
	if err := json.NewDecoder(resp.Body).Decode(&users); err != nil {
		handleError("Ошибка декодирования", err)
		return
	}

	printUsers(users)
}

func findUserByName() {
	fmt.Print("Введите имя для поиска: ")
	var name string
	fmt.Scan(&name)

	resp, err := http.Get(fmt.Sprintf("%s?name=%s", baseURL, name))
	if err != nil {
		handleError("Ошибка при получении данных:", err)
		return
	}

	var users []User
	if err := json.NewDecoder(resp.Body).Decode(&users); err != nil {
		handleError("Ошиька декодирвования", err)
		return
	}

	if len(users) == 0 {
		handleError("Пользователь не найден", nil)
	} else {
		printUsers(users)
	}
}

func createUser() {
	var user User
	fmt.Print("Введите имя: ")
	fmt.Scan(&user.Name)
	fmt.Print("Введите возраст: ")
	fmt.Scan(&user.Age)

	jsonData, err := json.Marshal(user)
	if err != nil {
		handleError("Ошибка преобразования ", err)
		return
	}

	resp, err := http.Post(baseURL, "application/json", bytes.NewBuffer(jsonData))
	if err != nil {
		handleError("Ошибка при создании", err)
		return
	}

	hadleResponse(resp, "Пользователь успешно создан")
}

func updateUser() {
	var id, name, age string
	fmt.Print("Введите ID для обновления: ")
	fmt.Scan(&id)
	fmt.Print("Введите новое имя: ")
	fmt.Scan(&name)
	fmt.Print("Введите новый возраст: ")
	fmt.Scan(&age)

	user := User{Name: name, Age: age}
	jsonData, err := json.Marshal(user)
	if err != nil {
		handleError("Ошибка декодирования", err)
		return
	}

	req, err := http.NewRequest(http.MethodPut, fmt.Sprintf("%s/%s", baseURL, id), bytes.NewBuffer(jsonData))
	if err != nil {
		fmt.Println("Ошибка создания запроса", err)
		return
	}
	req.Header.Set("Content-Type", "application/json")

	client := &http.Client{}
	resp, err := client.Do(req)
	if err != nil {
		handleError("Ошибка обновления", err)
		return
	}

	hadleResponse(resp, "Пользователь обновлен")
}

func deleteUser() {
	fmt.Print("Введите ID для удаления ")
	var id string
	fmt.Scan(&id)

	req, err := http.NewRequest(http.MethodDelete, fmt.Sprintf("%s/%s", baseURL, id), nil)

	if err != nil {
		handleError("Ошибка создания запроса", err)
		return
	}

	client := &http.Client{}
	resp, err := client.Do(req)
	if err != nil {
		handleError("Ошибка при удалении", err)
		return
	}

	hadleResponse(resp, "Пользователь успешно удален")
}
