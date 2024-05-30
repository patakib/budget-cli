package cmd

import (
	"database/sql"
	"log"
	"os"
	"time"

	_ "github.com/mattn/go-sqlite3"
	"github.com/spf13/cobra"
	"gopkg.in/yaml.v3"
)

type Category struct {
	Name   string `yaml:"name"`
	Amount int32  `yaml:"amount"`
}

type BasicConfig struct {
	Income            int32      `yaml:"income"`
	CategoriesPlanned []Category `yaml:"categories-planned"`
}

var InsertDate string
var InsertCategory string
var InsertAmount int32
var InsertComment string

func createDatabase(config BasicConfig) {
	os.Remove("./budget.db")
	db, err := sql.Open("sqlite3", "./budget.db")
	if err != nil {
		log.Fatal(err)
	}
	defer db.Close()

	createExpensesTable := `
		create table expenses (
			id integer not null primary key,
			date text,
			category text,
			amount integer,
			comment text
		);
	`
	createCategoriesTable := `
		create table categories (
			id integer not null primary key,
			name text,
			amount integer
		);
	`
	_, err = db.Exec(createExpensesTable)
	if err != nil {
		log.Printf("%q: %s\n", err, createExpensesTable)
		return
	}
	_, err = db.Exec(createCategoriesTable)
	if err != nil {
		log.Printf("%q: %s\n", err, createCategoriesTable)
		return
	}
	for _, element := range config.CategoriesPlanned {
		stmt, _ := db.Prepare("insert into categories(name, amount) values(?, ?)")
		stmt.Exec(element.Name, element.Amount)
	}
}

var CreateCmd = &cobra.Command{
	Use:   "create",
	Short: "Create new budget",
	Run: func(cmd *cobra.Command, args []string) {
		var config BasicConfig
		yamlFile, err := os.ReadFile("config.yaml")
		if err != nil {
			panic(err)
		}
		if err := yaml.Unmarshal(yamlFile, &config); err != nil {
			panic(err)
		}
		createDatabase(config)
	},
}

var AddCmd = &cobra.Command{
	Use:   "add",
	Short: "Add new expense",
	Long: `
		Add new expense with the following details: date, category, amount, comment.
		Example usage: 
			budget add --date 2024-06-05 --category car --amount 10000 --comment fuel
			or with short flags and using today by default:
			budget add -c car -a 10000 -m fuel
	`,

	Run: func(cmd *cobra.Command, args []string) {
		db, err := sql.Open("sqlite3", "./budget.db")
		if err != nil {
			log.Fatal(err)
		}
		defer db.Close()

		stmt, _ := db.Prepare("insert into expenses(date, category, amount, comment) values(?, ?, ?, ?)")
		stmt.Exec(InsertDate, InsertCategory, InsertAmount, InsertComment)
	},
}

func init() {
	AddCmd.Flags().StringVarP(&InsertDate, "date", "d", time.Now().Format("2006-01-02"), "Date of actual transaction in YYYY-MM-DD format")
	// TODO: check if valid
	AddCmd.Flags().StringVarP(&InsertCategory, "category", "c", "", "Category of transaction")
	AddCmd.Flags().Int32VarP(&InsertAmount, "amount", "a", 0, "Transaction amount")
	AddCmd.Flags().StringVarP(&InsertComment, "comment", "m", "", "Additional information")
}
