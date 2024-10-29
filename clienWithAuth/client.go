package main

import (
	"bytes"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"net/http"
	"os"
	"strings"
)

const baseURL = "http://localhost:8080/users"
const baseLogin = "http://localhost:8080/login"

type User struct {
	ID   string `json:"id,omitempty"`
	Name string `json:"name`
	Age  string `json:"age`
}

type UserResponse struct {
	Users []User `json:"users"`
}

type AuthResponse struct {
	Token string `json:"token"`
}

var sessionToken string

func main() {
	for {
		fmt.Println("\nВыберите действие:")
		fmt.Println("1. Войти в систему")
		fmt.Println("2. Показать всех пользователей")
		fmt.Println("3. Найти пользователся по имени")
		fmt.Println("4. Добавить нового пользователя")
		fmt.Println("5. Обновить данные пользователя")
		fmt.Println("6. Удалить пользователя")
		fmt.Println("0. Выход")

		var choice int
		fmt.Scan(&choice)

		switch choice {
		case 1:
			login()
		case 2:
			getUsers()
		case 3:
			findUserByName()
		case 4:
			createUser()
		case 5:
			updateUser()
		case 6:
			deleteUser()
		case 0:
			os.Exit(0)
		default:
			fmt.Println("Неккоректный ввод")
		}

	}
}

func login() {
	fmt.Print("Введите имя пользователя ")
	var username string
	fmt.Scan((&username))
	fmt.Print("Введите пароль ")
	var password string
	fmt.Scan(&password)
	password = strings.TrimSpace(password)
	h := sha256.New()
	h.Write([]byte(password))
	hashpass := hex.EncodeToString(h.Sum(nil))

	authData := map[string]string{"username": username, "password": strings.TrimSpace(hashpass)}
	jsonData, err := json.Marshal(authData)
	if err != nil {
		handleError("Ошибка преобразования", err)
		return
	}

	resp, err := http.Post(baseLogin, "application/json", bytes.NewBuffer(jsonData))
	if err != nil {
		handleError("Ошибка авторизации", err)
		return
	}
	defer resp.Body.Close()

	if resp.StatusCode == http.StatusOK {
		var autrResp AuthResponse
		if err := json.NewDecoder(resp.Body).Decode(&autrResp); err != nil {
			handleError("Ошибка декодирования ", err)
			return
		}
		sessionToken = autrResp.Token
		fmt.Println("Успешная авторизация токен сохранен")
	} else {
		handleError(fmt.Sprint("Ошибка авторизациии %s\n", resp.Body), nil)
	}

}

func createAuthRequest(method, url string, body *bytes.Buffer) (*http.Request, error) {
	req, err := http.NewRequest(method, url, body)
	if err != nil {
		return nil, err
	}
	req.Header.Set("Content-Type", "application/json")
	if sessionToken != "" {
		req.Header.Set("Authorization", "Bearer "+sessionToken)
	} else {
		fmt.Println("Отсутсвтует токен авторизации")
	}
	return req, nil
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
	req, err := createAuthRequest("GET", baseURL, bytes.NewBuffer(nil))
	if err != nil {
		handleError("Ошибка при создании запроса", err)
		return
	}
	client := &http.Client{}
	resp, err := client.Do(req)
	if err != nil {
		handleError("Ошибка при получении пользователей:", err)
		return
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		//handleError(fmt.Sprintf("Ошибка сервера: %d", resp.StatusCode), nil)
		return
	}

	var users []User
	if err := json.NewDecoder(resp.Body).Decode(&users); err != nil {
		handleError("Ошибка декодирвования", err)
		return
	}

	printUsers(users)
}

func findUserByName() {
	fmt.Print("Введите имя для поиска: ")
	var name string
	fmt.Scan(&name)

	req, err := createAuthRequest("GET", baseURL+"?name="+name, bytes.NewBuffer(nil))
	if err != nil {
		handleError("Ошибка при создании запроса", err)
	}
	client := &http.Client{}
	resp, err := client.Do(req)
	if err != nil {
		handleError("Ошибка при получении данных:", err)
		return
	}
	defer resp.Body.Close()

	var users []User
	if err := json.NewDecoder(resp.Body).Decode(&users); err != nil {
		handleError("Ошибка декодирвования", err)
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

	req, err := createAuthRequest("POST", baseURL, bytes.NewBuffer(jsonData))
	if err != nil {
		handleError("Ошибка при создании запроса", err)
		return
	}

	client := &http.Client{}
	resp, err := client.Do(req)
	if err != nil {
		handleError("Ошибка при отправке запроса", err)
		return
	}
	defer resp.Body.Close()

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

	req, err := createAuthRequest(http.MethodPut, fmt.Sprintf("%s/%s", baseURL, id), bytes.NewBuffer(jsonData))
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

	req, err := createAuthRequest(http.MethodDelete, fmt.Sprintf("%s/%s", baseURL, id), bytes.NewBuffer(nil))

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
