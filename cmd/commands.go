package cmd

import (
	"database/sql"
	"fmt"
	"log"
	"os"
	"path/filepath"
	"time"

	"github.com/jedib0t/go-pretty/v6/table"
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

var dateInput string
var categoryInput string
var categorySliceInput []string
var amountInput int32
var commentInput string
var fromInput string
var toInput string
var minInput int32
var maxInput int32

func getDatabasePath() string {
	// Get the path of the executable
	executablePath, err := os.Executable()
	if err != nil {
		log.Fatalf("Failed to get executable path: %v", err)
	}

	// Determine the directory of the executable
	executableDir := filepath.Dir(executablePath)

	// Construct the path to the database file
	dbFilePath := filepath.Join(executableDir, "budget.db")
	return dbFilePath
}

func getConfigPath() string {
	// Get the path of the executable
	executablePath, err := os.Executable()
	if err != nil {
		log.Fatalf("Failed to get executable path: %v", err)
	}

	// Determine the directory of the executable
	executableDir := filepath.Dir(executablePath)

	// Construct the path to the database file
	configFilePath := filepath.Join(executableDir, "config.yaml")
	return configFilePath
}

func createDatabase(config BasicConfig, dbFilePath string) {
	os.Remove(dbFilePath)
	db, err := sql.Open("sqlite3", dbFilePath)
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
	fmt.Println("Database and table successfully set up at:", dbFilePath)
}

func contains(slice []string, word string) bool {
	for _, w := range slice {
		if w == word {
			return true
		}
	}
	return false
}

var CreateCmd = &cobra.Command{
	Use:   "create",
	Short: "Create new budget",
	Run: func(cmd *cobra.Command, args []string) {
		var config BasicConfig
		configFilePath := getConfigPath()
		yamlFile, err := os.ReadFile(configFilePath)
		if err != nil {
			panic(err)
		}
		if err := yaml.Unmarshal(yamlFile, &config); err != nil {
			panic(err)
		}
		dbFilePath := getDatabasePath()
		createDatabase(config, dbFilePath)
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

	PreRunE: func(cmd *cobra.Command, args []string) error {

		// get database path
		dbFilePath := getDatabasePath()

		// handle date input error
		_, err := time.Parse("2006-01-02", dateInput)
		if err != nil {
			return fmt.Errorf("your date input couldn't be handled as date: %v", dateInput)
		}

		// handle category input error
		// initialize db connection
		db, err := sql.Open("sqlite3", dbFilePath)
		if err != nil {
			log.Fatal(err)
		}
		defer db.Close()

		// read existing categories
		rows, err := db.Query(`
			select name from categories;
			`)
		if err != nil {
			log.Fatal(err)
		}
		defer rows.Close()

		// iterating through categories
		t := table.NewWriter()
		t.SetOutputMirror(os.Stdout)
		var match int8 = 0
		for rows.Next() {
			var categoryName string
			err = rows.Scan(&categoryName)
			if err != nil {
				log.Fatal(err)
			}
			t.AppendRow([]interface{}{categoryName})
			if categoryName == categoryInput {
				match++
			}
		}

		err = rows.Err()
		if err != nil {
			log.Fatal(err)
		}
		if match == 0 {
			return fmt.Errorf("your category input (%v) does not match with any existing expense categories. Available categories: \n%s", categoryInput, t.Render())
		}
		return nil
	},

	Run: func(cmd *cobra.Command, args []string) {
		// get database file path
		dbFilePath := getDatabasePath()
		db, err := sql.Open("sqlite3", dbFilePath)
		if err != nil {
			log.Fatal(err)
		}
		defer db.Close()
		stmt, _ := db.Prepare("insert into expenses(date, category, amount, comment) values(?, ?, ?, ?)")
		stmt.Exec(dateInput, categoryInput, amountInput, commentInput)
	},
}

var FilterCmd = &cobra.Command{
	Use:   "filter",
	Short: "Filter budget data",
	Long: `
	Filter budget data by multiple criteria, usage:
	budget filter --from 2024-01-01 --to 2024-02-28 --category groceries --max 100000 --min 10000
	`,

	//TODO PreRunE

	Run: func(cmd *cobra.Command, args []string) {
		dbFilePath := getDatabasePath()
		db, err := sql.Open("sqlite3", dbFilePath)
		if err != nil {
			log.Fatal(err)
		}
		defer db.Close()
		stmt, _ := db.Prepare(`
			select category, date, amount, comment from expenses
			where 
			date >= ? 
			and date <= ?
			and amount >= ?
			and amount <= ?
			order by date asc
		`)
		rows, err := stmt.Query(fromInput, toInput, minInput, maxInput)
		if err != nil {
			log.Fatal(err)
		}
		defer rows.Close()

		// initialize table for pretty print
		t := table.NewWriter()
		t.SetOutputMirror(os.Stdout)
		t.AppendHeader(table.Row{"Category", "Date", "Amount", "Comment"})

		// iterating through records and add to table as row
		var filteredSum int32
		for rows.Next() {
			var categoryName string
			var date string
			var amount int32
			var comment string
			err = rows.Scan(&categoryName, &date, &amount, &comment)
			if err != nil {
				log.Fatal(err)
			}
			if contains(categorySliceInput, categoryName) || categorySliceInput == nil {
				filteredSum += amount
				t.AppendRow([]interface{}{categoryName, date, amount, comment})
			}
			err = rows.Err()
			if err != nil {
				log.Fatal(err)
			}
		}
		// add footer to table, set style of table and render
		t.AppendFooter(table.Row{"TOTAL", "", filteredSum, ""})
		t.SetStyle(table.StyleColoredBright)
		t.Render()
	},
}

var StatusCmd = &cobra.Command{
	Use:   "status",
	Short: "Provide actual status of your expenses",
	Long: `
	Provide status info about expenses.
	It compares your budget at current stage against your monthly plans.
	`,

	PreRunE: func(cmd *cobra.Command, args []string) error {
		// check whether there is at least one expense

		// get database path
		dbFilePath := getDatabasePath()

		//initialize db connection
		db, err := sql.Open("sqlite3", dbFilePath)
		if err != nil {
			log.Fatal(err)
		}
		defer db.Close()

		//get expenses
		var countExpenses int32
		err = db.QueryRow(`
		select count(id) from expenses
		where strftime("%Y", date("now"))=strftime("%Y", date)
		and strftime("%m", date("now"))=strftime("%m", date);
		`).Scan(&countExpenses)
		if err != nil {
			log.Fatal(err)
		}
		if countExpenses == 0 {
			return fmt.Errorf("you don't have any registered expenses for the current month")
		}
		return nil
	},

	Run: func(cmd *cobra.Command, args []string) {
		// get database file path
		dbFilePath := getDatabasePath()

		db, err := sql.Open("sqlite3", dbFilePath)
		if err != nil {
			log.Fatal(err)
		}
		defer db.Close()

		//get default category sum per month
		var categorySumDefault int32
		err = db.QueryRow("select sum(amount) from categories;").Scan(&categorySumDefault)
		if err != nil {
			log.Fatal(err)
		}

		//get current month expenses
		var categorySumCurrent int32
		err = db.QueryRow(`
			select sum(amount) from expenses 
			where strftime("%Y", date("now"))=strftime("%Y", date)
			and strftime("%m", date("now"))=strftime("%m", date);
		`).Scan(&categorySumCurrent)
		if err != nil {
			log.Fatal(err)
		}

		//get all category defaults
		rows, err := db.Query(`
			with grouped_expenses as (
				select category, sum(amount) as amount from expenses
				where strftime("%Y", date("now"))=strftime("%Y", date)
				and strftime("%m", date("now"))=strftime("%m", date)
				group by category
				order by category asc
			)
			select c.name, c.amount as planned, ifnull(e.amount, 0) as fact, ifnull(c.amount-e.amount, 0) as balance 
			from categories c 
			left join grouped_expenses e
			on c.name=e.category
			order by c.name asc;
			`)
		if err != nil {
			log.Fatal(err)
		}
		defer rows.Close()

		// initialize table for pretty print
		t := table.NewWriter()
		t.SetOutputMirror(os.Stdout)
		t.AppendHeader(table.Row{"Category", "Planned Monthly Expense", "Actual Expense this Month", "Balance"})

		// iterating through records and add to table as row
		for rows.Next() {
			var categoryName string
			var plannedAmount int32
			var factAmount int32
			var balance int32
			err = rows.Scan(&categoryName, &plannedAmount, &factAmount, &balance)
			if err != nil {
				log.Fatal(err)
			}
			t.AppendRow([]interface{}{categoryName, plannedAmount, factAmount, balance})
		}
		err = rows.Err()
		if err != nil {
			log.Fatal(err)
		}

		// add footer to table with totals, set style of table and render
		t.AppendFooter(table.Row{"TOTAL", categorySumDefault, categorySumCurrent, categorySumDefault - categorySumCurrent})
		t.SetStyle(table.StyleColoredBright)
		t.Render()

	},
}

func init() {
	now := time.Now()
	currentYear, currentMonth, _ := now.Date()
	currentLocation := now.Location()
	firstOfMonth := time.Date(currentYear, currentMonth, 1, 0, 0, 0, 0, currentLocation).Format("2006-01-02")

	AddCmd.Flags().StringVarP(&dateInput, "date", "d", time.Now().Format("2006-01-02"), "Date of actual transaction in YYYY-MM-DD format")
	AddCmd.Flags().StringVarP(&categoryInput, "category", "c", "", "Category of transaction")
	AddCmd.Flags().Int32VarP(&amountInput, "amount", "a", 0, "Transaction amount")
	AddCmd.Flags().StringVarP(&commentInput, "comment", "m", "", "Additional information")
	FilterCmd.Flags().StringSliceVarP(&categorySliceInput, "category", "c", nil, "Categories to filter for")
	FilterCmd.Flags().Int32VarP(&minInput, "min", "", 0, "Min transaction amount, default: 0")
	FilterCmd.Flags().Int32VarP(&maxInput, "max", "", 100000, "Max transaction amount, default: 100.000")
	FilterCmd.Flags().StringVarP(&fromInput, "from", "f", firstOfMonth, "From date in YYYY-MM-DD format, default: first day of current month")
	FilterCmd.Flags().StringVarP(&toInput, "to", "t", time.Now().Format("2006-01-02"), "To date in YYYY-MM-DD format, default: today")
}
