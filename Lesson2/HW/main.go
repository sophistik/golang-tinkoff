package main

import (
	"bufio"
	"encoding/csv"
	"encoding/json"
	"github.com/pkg/errors"
	"io"
	"io/ioutil"
	"log"
	"os"
)

type Users struct {
	Nick        string        `json:"Nick"`
	Email       string        `json:"Email"`
	CreatedAt   string        `json:"Created_at"`
	Subscribers []Subscribers `json:"Subscribers"`
}

type Subscribers struct {
	Email     string `json:"Email"`
	CreatedAt string `json:"Created_at"`
}

type result struct {
	ID   int           `json:"id"`
	From string        `json:"from"`
	To   string        `json:"to"`
	Path []Subscribers `json:"path,omitempty"`
}

type queue []string

func (q *queue) push(v string) {
	*q = append(*q, v)
}

func (q *queue) pop() {
	*q = append((*q)[:0], (*q)[1:]...)
}

func (q *queue) top() string {
	return (*q)[0]
}

func (q *queue) empty() bool {
	return len(*q) == 0
}

func readUsers(path string) ([]Users, error) {
	jsonFile, err := os.Open(path)
	if err != nil {
		return nil, errors.Wrap(err, "can't open file")
	}
	defer jsonFile.Close()

	byteValue, err := ioutil.ReadAll(jsonFile)
	if err != nil {
		return nil, errors.Wrap(err, "can't read from json")
	}

	var users []Users
	err = json.Unmarshal(byteValue, &users)
	if err != nil {
		return nil, errors.Wrap(err, "can't unmarshal file")
	}

	return users, err
}

func prepareInfoForBST(users []Users) (map[string][]string, map[string]string) {
	friends := make(map[string][]string)
	createdAt := make(map[string]string)

	for _, user := range users {
		for _, subscriber := range user.Subscribers {
			_, ok := friends[subscriber.Email]
			if !ok {
				friends[subscriber.Email] = make([]string, 0)
			}

			friends[subscriber.Email] = append(friends[subscriber.Email], user.Email)
			createdAt[subscriber.Email] = subscriber.CreatedAt
		}
	}

	return friends, createdAt
}

func findPaths(inputFile string, friends map[string][]string, createdAt map[string]string) ([]result, error) {
	csvInput, err := os.Open(inputFile)
	if err != nil {
		return nil, errors.Wrap(err, "can't open CSV file")
	}

	defer csvInput.Close()

	input := csv.NewReader(bufio.NewReader(csvInput))
	id := 1
	output := make([]result, 0)

	for {
		line, err := input.Read()
		if err != nil {
			if err == io.EOF {
				break
			}

			return nil, errors.Wrap(err, "can't read CSV file")
		}

		from := line[0]
		to := line[1]
		pathFromBFS := bfs(from, to, friends)
		path := make([]Subscribers, len(pathFromBFS))

		for i, subscriber := range pathFromBFS {
			path[i] = Subscribers{Email: subscriber, CreatedAt: createdAt[subscriber]}
		}

		curResult := result{ID: id, From: from, To: to, Path: path}
		output = append(output, curResult)
		id++
	}

	return output, nil
}

func bfs(from, to string, friends map[string][]string) []string {
	var q queue

	path := make(map[string][]string)
	path[from] = make([]string, 0)

	q.push(from)

	for !q.empty() {
		currUser := q.top()

		if currUser == to {
			return path[currUser]
		}

		currUserFriends := friends[currUser]

		for _, friend := range currUserFriends {
			_, ok := path[friend]
			if !ok {
				path[friend] = make([]string, len(path[currUser]))
				copy(path[friend], path[currUser])

				if currUser != from {
					path[friend] = append(path[friend], currUser)
				}

				q.push(friend)
			}
		}

		q.pop()
	}

	return nil
}

func writeToJSON(outputFile string, outputData []result) error {
	sample, err := json.MarshalIndent(outputData, "", "      ")
	if err != nil {
		return errors.Wrap(err, "can't marshal some output data into JSON")
	}

	file, err := os.Create(outputFile)
	if err != nil {
		return errors.Wrap(err, "can't create a file")
	}

	defer file.Close()

	writer := bufio.NewWriter(file)

	defer writer.Flush()

	_, err = writer.Write(sample)
	if err != nil {
		return errors.Wrap(err, "can't write a result into JSON file")
	}

	return nil
}

func main() {
	users, err := readUsers("users.json")
	if err != nil {
		log.Fatal("can't read JSON file ", err)
	}

	friends, createdAt := prepareInfoForBST(users)
	output, err := findPaths("input.csv", friends, createdAt)
	if err != nil {
		log.Fatal(err)
	}

	err = writeToJSON("result.json", output)
	if err != nil {
		log.Fatal("can't write a result:", err)
	}
}
