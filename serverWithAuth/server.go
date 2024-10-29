package main

import (
	"context"
	"encoding/json"
	"fmt"
	"log"
	"net/http"
	"os"
	"strconv"
	"strings"
	"sync"
	"time"

	"github.com/dgrijalva/jwt-go"
	"github.com/gorilla/mux"
	"go.mongodb.org/mongo-driver/bson"
	"go.mongodb.org/mongo-driver/bson/primitive"
	"go.mongodb.org/mongo-driver/mongo"
	"go.mongodb.org/mongo-driver/mongo/options"
)

// для многоклиентности
var (
	client *mongo.Client
	users  = []User{}
	mu     sync.Mutex
)

func initDatabase() {
	var err error
	client, err = mongo.Connect(context.TODO(), options.Client().ApplyURI("mongodb://localhost:27012"))
	if err != nil {
		log.Fatal(err)
	}

	err = client.Ping(context.TODO(), nil)
	if err != nil {
		log.Fatal(err)
	}
	log.Println("Подключение к бд успешно")
}

type Credentials struct {
	Username string `json:"username"`
	Password string `json:"password"`
}

type UserData struct {
	Username string `bson:"username"`
	Password string `bson:"password"`
}

var jwtKey = []byte("key")

func generateJWT(username string) (string, error) {
	fmt.Println("Сгенерирован токен")
	expirationTime := time.Now().Add(30 * time.Minute)
	claims := &jwt.StandardClaims{
		Subject:   username,
		ExpiresAt: expirationTime.Unix(),
	}

	token := jwt.NewWithClaims(jwt.SigningMethodHS256, claims)
	return token.SignedString(jwtKey)
}

func authMiddleware(next http.Handler) http.Handler {
	fmt.Println("аутентификация")
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		tokenString := r.Header.Get("Authorization")
		fmt.Println("Токен авторизации: ", tokenString)
		if tokenString == "" {
			fmt.Println("Нет токена авторизации")
			handleError(w, "Нет токена авторизации", http.StatusUnauthorized)
			return
		}

		// Убираем префикс "Bearer " из строки токена
		tokenString = strings.TrimPrefix(tokenString, "Bearer ")
		fmt.Println("Токен после удаления префикса: ", tokenString)

		claims := &jwt.StandardClaims{}
		token, err := jwt.ParseWithClaims(tokenString, claims, func(token *jwt.Token) (interface{}, error) {
			fmt.Println("Создаем jwt")
			return jwtKey, nil
		})

		if err != nil || !token.Valid {
			handleError(w, "Неверный токен", http.StatusUnauthorized)
			return
		}

		r = r.WithContext(r.Context())
		fmt.Println("Авторизация прошла, запрос пропущен")
		next.ServeHTTP(w, r)
	})
}

func loginHandler(w http.ResponseWriter, r *http.Request) {
	fmt.Println("Вход в систему")
	var creds Credentials
	if err := json.NewDecoder(r.Body).Decode(&creds); err != nil {
		handleError(w, "Неверные данные запроса", http.StatusBadRequest)
		return
	}
	fmt.Println("Cred=", creds)
	var userdata UserData
	collection := client.Database("lab8").Collection("login")
	err := collection.FindOne(context.TODO(), bson.M{"login": creds.Username}).Decode(&userdata)
	fmt.Println("Userdata=", userdata)

	if err != nil {
		if err == mongo.ErrNoDocuments {
			handleError(w, "Неверное имя пользователя или пароль", http.StatusUnauthorized)
		} else {
			handleError(w, "Ошибка при запросе к бд", http.StatusInternalServerError)
		}
		return
	}

	// fmt.Println("2")
	// h := strings.TrimSpace(creds.Password)
	// hs := sha256.New()
	// hs.Write([]byte(h))
	// hexhash := hex.EncodeToString(hs.Sum(nil))

	if userdata.Password != creds.Password {
		handleError(w, "Неверное имя пользователя или пароль", http.StatusUnauthorized)
		return
	}

	token, err := generateJWT(creds.Username)
	if err != nil {
		handleError(w, "Не удалось создать токен", http.StatusInternalServerError)
		return
	}
	fmt.Println("3")
	w.Header().Set("Content-type", "application/json")
	json.NewEncoder(w).Encode(map[string]string{"token": token})
}

type User struct {
	ID   primitive.ObjectID `json:"id,omitempty" bson:"_id,omitempy"`
	Name string             `json:"name" bson:"name"`
	Age  string             `json:"age" bson:"age"`
}

//var client *mongo.Client

func connectDB() {
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()
	var err error
	client, err = mongo.Connect(ctx, options.Client().ApplyURI("mongodb://localhost:27017"))
	if err != nil {
		log.Fatal(err)
	}
	fmt.Println("Подключение к бд успешно")
}

func handleError(w http.ResponseWriter, message string, code int) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(code)
	json.NewEncoder(w).Encode(map[string]string{"error": message})
}

func validateUser(user User) (bool, string) {
	if strings.TrimSpace(user.Name) == "" {
		return false, "Имя не может быть пустым"
	}
	return true, ""
}

func getUsers(w http.ResponseWriter, r *http.Request) {
	mu.Lock()
	defer mu.Unlock()
	fmt.Println("Вызвано getUsers")
	w.Header().Set("Content-Type", "application/json")

	name := r.URL.Query().Get("name")
	minAgeParam := r.URL.Query().Get("min_age")
	maxAgeParam := r.URL.Query().Get("max_age")
	limitParam := r.URL.Query().Get("limit")
	pageParam := r.URL.Query().Get("page")

	var minAge, maxAge, limit, page int
	var err error

	if limitParam != "" {
		limit, err = strconv.Atoi(limitParam)
		if err != nil || limit <= 0 {
			handleError(w, "Неверное значение limit", http.StatusBadRequest)
			return
		}
	} else {
		limit = 10
	}

	if pageParam != "" {
		page, err := strconv.Atoi(pageParam)
		if err != nil || page <= 0 {
			handleError(w, "Неверное значение page", http.StatusBadRequest)
			return
		}
	} else {
		page = 1
	}

	if minAgeParam != "" {
		minAge, err = strconv.Atoi(minAgeParam)
		if err != nil {
			handleError(w, "Неверное значение min_age", http.StatusBadRequest)
			return
		}
	}

	if maxAgeParam != "" {
		maxAge, err = strconv.Atoi(maxAgeParam)
		if err != nil {
			handleError(w, "Неверное значение max_age", http.StatusBadRequest)
			return
		}
	}

	//смещение
	skip := (page - 1) * limit

	var users []User
	collection := client.Database("lab8").Collection("test")
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	filter := bson.M{}
	if name != "" {
		fmt.Println("name= ", name)
		filter["name"] = bson.M{"$regex": name, "$options": "i"}
	}

	if minAge > 0 || maxAge > 0 {
		ageFilter := bson.M{}
		if minAge > 0 {
			ageFilter["$gte"] = minAge
		}
		if maxAge > 0 {
			ageFilter["$lte"] = maxAge
		}
		filter["age"] = ageFilter
	}

	findOptions := options.Find()
	findOptions.SetSkip(int64(skip))
	findOptions.SetLimit(int64(limit))

	cur, err := collection.Find(ctx, filter, findOptions)
	if err != nil {
		handleError(w, "Ошибка чтения из бд", http.StatusInternalServerError)
		return
	}
	defer cur.Close(ctx)

	// totalUsers, err := collection.CountDocuments(ctx, filter)
	// if err != nil {
	// 	handleError(w, "Ошибка при подсчете данных", http.StatusInternalServerError)
	// 	return
	// }

	for cur.Next(ctx) {
		var user User
		err := cur.Decode(&user)
		if err != nil {
			handleError(w, "Ошибка обработки данных", http.StatusInternalServerError)
			return
		}
		users = append(users, user)
	}

	if err := cur.Err(); err != nil {
		handleError(w, "Ошибка чтения из бд", http.StatusInternalServerError)
		return
	}
	//fmt.Println(totalUsers)
	//totalPages := (int(totalUsers) + limit - 1) / limit

	// json.NewEncoder(w).Encode(map[string]interface{}{
	// 	"users":        users,
	// 	"total_pages":  totalPages,
	// 	"total_users":  totalUsers,
	// 	"current_page": page,
	// })
	fmt.Println("Отправляем пользователей:", users)
	jsonData, _ := json.Marshal(users)
	fmt.Println("Ответ", string(jsonData))
	json.NewEncoder(w).Encode(users)
}

func getUser(w http.ResponseWriter, r *http.Request) {
	fmt.Println("Вызвано getUser")
	w.Header().Set("Content-Type", "application/json")
	params := mux.Vars(r)
	id := params["id"]

	var user User
	colletion := client.Database("lab8").Collection("test")
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	err := colletion.FindOne(ctx, bson.M{"_id": id}).Decode(&user)
	if err != nil {
		handleError(w, "Пользователь не найдем", http.StatusNotFound)
		return
	}

	json.NewEncoder(w).Encode(user)
}

func createUser(w http.ResponseWriter, r *http.Request) {
	mu.Lock()
	defer mu.Unlock()
	fmt.Println("Вызвано createUsers")
	w.Header().Set("Content-Type", "application/json")
	var newUser User
	err := json.NewDecoder(r.Body).Decode(&newUser)
	if err != nil {
		handleError(w, "Неправильные данные", http.StatusBadRequest)
		return
	}

	if valid, message := validateUser(newUser); !valid {
		handleError(w, message, http.StatusBadRequest)
		return
	}

	collection := client.Database("lab8").Collection("test")
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	newUser.ID = primitive.NewObjectID()
	_, err = collection.InsertOne(ctx, newUser)
	if err != nil {
		handleError(w, "Ошибка при добавлении пользователя", http.StatusInternalServerError)
		return
	}

	json.NewEncoder(w).Encode(newUser)
}

func updateUser(w http.ResponseWriter, r *http.Request) {
	fmt.Println("Вызвано updateUsers")
	w.Header().Set("Content-Type", "application/json")
	params := mux.Vars(r)
	id := params["id"]
	objectId, err := primitive.ObjectIDFromHex(id)
	if err != nil {
		handleError(w, "Неправильный ID", http.StatusBadRequest)
		return
	}

	var updatedUser User
	err = json.NewDecoder(r.Body).Decode(&updatedUser)
	if err != nil {
		handleError(w, "Неправильные данные", http.StatusBadRequest)
		return
	}

	if valid, message := validateUser(updatedUser); !valid {
		handleError(w, message, http.StatusBadRequest)
		return
	}

	collection := client.Database("lab8").Collection("test")
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	filter := bson.M{"_id": objectId}
	update := bson.M{
		"$set": bson.M{
			"name": updatedUser.Name,
			"age":  updatedUser.Age,
		},
	}

	_, err = collection.UpdateOne(ctx, filter, update)
	if err != nil {
		handleError(w, "Ошибка при обновлении данных", http.StatusInternalServerError)
	}

	json.NewEncoder(w).Encode(map[string]string{"message": "Пользователь обновлен"})
}

func deleteUser(w http.ResponseWriter, r *http.Request) {
	fmt.Println("Вызвано deleteUsers")
	w.Header().Set("Content-Type", "application/json")
	params := mux.Vars(r)
	id := params["id"]

	objectId, err := primitive.ObjectIDFromHex(id)
	if err != nil {
		handleError(w, "Неправильный ID", http.StatusBadRequest)
		return
	}

	collection := client.Database("lab8").Collection("test")
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	result, err := collection.DeleteOne(ctx, bson.M{"_id": objectId})
	if err != nil {
		handleError(w, "Ошибка при удалении пользователя", http.StatusInternalServerError)
		return
	}

	if result.DeletedCount == 0 {
		handleError(w, "Пользователь не найден", http.StatusNotFound)
		return
	}

	json.NewEncoder(w).Encode(map[string]string{"message": "Пользователь удален"})

}

func main() {
	connectDB()
	r := mux.NewRouter()
	r.HandleFunc("/login", loginHandler).Methods("POST")

	protected := r.PathPrefix("/users").Subrouter()
	protected.Use(authMiddleware)
	protected.HandleFunc("", getUsers).Methods("GET")
	protected.HandleFunc("/{id}", getUser).Methods("GET")
	protected.HandleFunc("", createUser).Methods("POST")
	protected.HandleFunc("/{id}", updateUser).Methods("PUT")
	protected.HandleFunc("/{id}", deleteUser).Methods("DELETE")

	fmt.Println("Сервер запущен на порту 8080")
	if err := http.ListenAndServe(":8080", r); err != nil {
		fmt.Println("Ошибка запуска")
		os.Exit(1)
	}

}
