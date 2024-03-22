package main

import (
	"bufio"
	"fmt"
	"html/template"
	"math/rand"
	"net/http"
	"os"
	"strconv"
	"strings"
	"time"
)

type Page struct {
	Title      string
	Question   Question
	Score      int
	Difficulty string
}

type Question struct {
	Text       string
	Options    []string
	CorrectAns string
}

var questions []Question
var currentQuestionIndex int
var score int

func loadQuestions(filename string) ([]Question, error) {
	file, err := os.Open(filename)
	if err != nil {
		return nil, err
	}
	defer file.Close()

	var questions []Question
	var q Question
	scanner := bufio.NewScanner(file)
	for scanner.Scan() {
		line := scanner.Text()
		if line == "" {
			shuffleOptions(&q)
			questions = append(questions, q)
			q = Question{}
			continue
		}
		parts := strings.SplitN(line, ":", 2)
		if len(parts) != 2 {
			return nil, fmt.Errorf("ligne invalide dans le fichier: %s", line)
		}
		switch parts[0] {
		case "text":
			q.Text = parts[1]
		case "option":
			q.Options = append(q.Options, parts[1])
		case "answer":
			q.CorrectAns = parts[1]
		}
	}
	if err := scanner.Err(); err != nil {
		return nil, err
	}

	if len(q.Text) > 0 {
		questions = append(questions, q)
	}

	return questions, nil
}

func shuffleQuestions(questions []Question) {
	r := rand.New(rand.NewSource(time.Now().UnixNano()))
	r.Shuffle(len(questions), func(i, j int) {
		questions[i], questions[j] = questions[j], questions[i]
	})
}

func shuffleOptions(q *Question) {
	r := rand.New(rand.NewSource(time.Now().UnixNano()))
	r.Shuffle(len(q.Options), func(i, j int) {
		q.Options[i], q.Options[j] = q.Options[j], q.Options[i]
	})
}

func indexHandler(w http.ResponseWriter, r *http.Request) {
	score = 0
	http.ServeFile(w, r, "./src/html/index.html")
}

func levelHandler(w http.ResponseWriter, r *http.Request) {
	http.ServeFile(w, r, "./src/html/level.html")
}

func setDifficultyCookie(w http.ResponseWriter, difficulty string) {
	expiration := time.Now().Add(24 * time.Hour)
	cookie := http.Cookie{Name: "difficulty", Value: difficulty, Expires: expiration}
	http.SetCookie(w, &cookie)
}

func quizHandler(w http.ResponseWriter, r *http.Request) {
	score = 0
	difficulty := r.URL.Query().Get("difficulty")
	setDifficultyCookie(w, difficulty)

	var questionsFilename string
	switch difficulty {
	case "facile":
		questionsFilename = "./src/questions/questions_easy.txt"
	case "average":
		questionsFilename = "./src/questions/questions_average.txt"
	case "hard":
		questionsFilename = "./src/questions/questions_hard.txt"
	default:
		http.Error(w, "Niveau de difficulté invalide", http.StatusBadRequest)
		return
	}

	var err error
	questions, err = loadQuestions(questionsFilename)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	shuffleQuestions(questions)
	currentQuestionIndex = 0

	http.Redirect(w, r, "/quiz-page", http.StatusSeeOther)
}

func quizPageHandler(w http.ResponseWriter, r *http.Request) {
	if len(questions) == 0 || currentQuestionIndex >= len(questions) {
		http.Error(w, "No questions available or index out of range", http.StatusInternalServerError)
		return
	}
	question := questions[currentQuestionIndex]

	page := Page{
		Title:    "Quiz de Pâques",
		Question: question,
	}

	tmpl, err := template.ParseFiles("./src/html/quiz.html")
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	err = tmpl.Execute(w, page)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
}

func answerHandler(w http.ResponseWriter, r *http.Request) {
	r.ParseForm()
	userAnswer := r.Form.Get("answer")

	if userAnswer == "" {
		http.Error(w, "Veuillez sélectionner une réponse", http.StatusBadRequest)
		return
	}

	correctAnswer := questions[currentQuestionIndex].CorrectAns

	var message string
	var resultClass string
	var correct bool

	if userAnswer == correctAnswer {
		score++
		message = "Correct! Well done!"
		correct = true
	} else {
		message = "Incorrect. Try again!"
		resultClass = "incorrect"
	}

	lastQuestionIndex := len(questions) - 1
	lastQuestionCorrect := currentQuestionIndex == lastQuestionIndex && userAnswer == correctAnswer

	fmt.Println("lastQuestionCorrect", lastQuestionCorrect)

	if lastQuestionCorrect {
		http.Redirect(w, r, "/score?score="+strconv.Itoa(score)+"&difficulty="+r.URL.Query().Get("difficulty"), http.StatusSeeOther)
		return
	} else if currentQuestionIndex == len(questions)-1 {
		http.Redirect(w, r, "/score?score="+strconv.Itoa(score)+"&difficulty="+r.URL.Query().Get("difficulty"), http.StatusSeeOther)
		return
	}

	resultPageData := struct {
		Message       string
		Correct       bool
		CorrectAnswer string
		ResultClass   string
		LastQuestion  bool
	}{
		Message:       message,
		Correct:       correct,
		CorrectAnswer: correctAnswer,
		ResultClass:   resultClass,
		LastQuestion:  lastQuestionCorrect,
	}

	tmpl, err := template.ParseFiles("./src/html/result.html")
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
	err = tmpl.Execute(w, resultPageData)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	currentQuestionIndex++

	http.Redirect(w, r, "/quiz-page", http.StatusSeeOther)
}

func restartQuizHandler(w http.ResponseWriter, r *http.Request) {
	cookie, err := r.Cookie("difficulty")
	if err != nil {
		http.Error(w, "Impossible de récupérer la difficulté", http.StatusBadRequest)
		return
	}

	difficulty := cookie.Value
	http.Redirect(w, r, "/quiz?difficulty="+difficulty, http.StatusSeeOther)
}

func scoreHandler(w http.ResponseWriter, r *http.Request) {
	scoreStr := r.URL.Query().Get("score")
	score, err := strconv.Atoi(scoreStr)
	if err != nil {
		http.Error(w, "Score invalide", http.StatusBadRequest)
		return
	}

	difficulty := r.FormValue("difficulty")

	page := Page{
		Title:      "Score Final",
		Score:      score,
		Difficulty: difficulty,
	}

	tmpl, err := template.ParseFiles("./src/html/score.html")
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	err = tmpl.Execute(w, page)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
}

func main() {
	http.HandleFunc("/", indexHandler)
	http.HandleFunc("/level", levelHandler)
	http.HandleFunc("/quiz", quizHandler)
	http.HandleFunc("/quiz-page", quizPageHandler)
	http.HandleFunc("/answer", answerHandler)
	http.HandleFunc("/score", scoreHandler)
	http.HandleFunc("/restart", restartQuizHandler)
	fmt.Println("Server is running on: http://localhost:8080")
	http.ListenAndServe(":8080", nil)
}
