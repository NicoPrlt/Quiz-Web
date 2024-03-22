BINARY_NAME := QUIZ.exe

	SRCS :=     ./src/main.go\

build:
	@echo "Construction du projet..."
		go build -o $(BINARY_NAME) $(SRCS)

clean:
	@echo "Nettoyage..."
		del /F $(BINARY_NAME)

run: build
	@echo "Execution du programme..."
		./$(BINARY_NAME)


	.PHONY: build clean run