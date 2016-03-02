package main

import (
	"database/sql"
	"fmt"
	"html/template"
	"log"
	"net/http"

	_ "github.com/go-sql-driver/mysql"
)

const dbHost = "mysql-service:3306"
const dbUser = "root"
const dbPassword = "password"
const dbName = "todo"

func main() {
	http.HandleFunc("/", todoList)
	http.HandleFunc("/save", saveItem)
	log.Fatal(http.ListenAndServe(":3000", nil))
}

// Todo represents a single 'todo', or item.
type Todo struct {
	ID   int
	Item string
}

// todoList shows the todo list along with the form to add a new item to the
// list.
func todoList(w http.ResponseWriter, r *http.Request) {
	tmpl := `
<html>
	<head>
		<title>List of items</title>
	</head>
	<body>
		<h1>Todo list:</h1>
		<form action="/save" method="post">
			Add item: <input type="text" name="item" /><br />
			<input type="submit" value="Add" /><br /><hr />
		</form>
		<ul>
		{{range $key, $value := .TodoItems}}
			<li>{{ $value.Item }}</li>
		{{end}}
		</ul>
	</body>
</html>
`
	t, err := template.New("todolist").Parse(tmpl)
	if err != nil {
		log.Fatal(err)
	}

	dbc := db()
	rows, err := dbc.Query("SELECT * FROM items")
	if err != nil {
		log.Fatal(err)
	}

	todos := []Todo{}
	for rows.Next() {
		todo := Todo{}
		err = rows.Scan(&todo.ID, &todo.Item)
		todos = append(todos, todo)
	}

	data := struct {
		TodoItems []Todo
	}{
		TodoItems: todos,
	}

	t.Execute(w, data)
}

// saveItem saves a new todo item and then redirects the user back to the list
func saveItem(w http.ResponseWriter, r *http.Request) {
	dbc := db()
	stmt, err := dbc.Prepare("INSERT items SET item=?")
	if err != nil {
		log.Fatal(err)
	}
	_, err = stmt.Exec(r.FormValue("item"))
	if err != nil {
		log.Fatal(err)
	}
	http.Redirect(w, r, "/", 301)
}

// db creates a connection to the database and creates the items table if it
// does not already exist.
func db() *sql.DB {
	connStr := fmt.Sprintf(
		"%s:%s@tcp(%s)/%s?charset=utf8&parseTime=True&loc=Local",
		dbUser,
		dbPassword,
		dbHost,
		dbName,
	)
	db, err := sql.Open("mysql", connStr)
	if err != nil {
		log.Fatal(err)
	}
	_, err = db.Exec(`CREATE TABLE IF NOT EXISTS items(
		id integer NOT NULL AUTO_INCREMENT,
		item varchar(255),
		PRIMARY KEY (id)
	)`)
	return db
}
