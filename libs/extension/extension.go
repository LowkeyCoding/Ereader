package extension

import (
	"database/sql"
	"encoding/json"
	"fmt"

	"github.com/dgrijalva/jwt-go"
	"github.com/gofiber/fiber"
)

// < ----- Extension Generators ----- >

//Extension descripes the structure of an extension.
type Extension struct {
	Name           string          `json:"Name"`
	Views          []View          `json:"Views"`
	DatabaseTables []DatabaseTable `json:"DatabaseTable"`
}

// LoadExtension loads a given extension.
func (extension *Extension) LoadExtension() error {
	return nil
}

// View descripes the structure of an view.
type View struct {
	Path               string        `json:"Path"`               // The path the view will be rendered to.
	ViewPath           string        `json:"ViewPath"`           // The view name. Will be used to select the correct view to render.
	NeedsQuerying      bool          `json:"needsQuerying"`      // The flag to enable querying
	QueryVariableNames []string      `json:"QueryVariableNames"` // Contains a a list of variable names used in the DatabaseQuery if it is set.
	DatabaseQuery      DatabaseQuery `json:"DatabaseQueries"`    // The result will be passed to the tempalte generator.
}

// GenerateView generates a view based on the view structure.
func (view *View) GenerateView(app *fiber.App, DB *sql.DB) {
	app.Get(view.Path, func(c *fiber.Ctx) {
		// Get current user information from the claims map.
		bind := fiber.Map{}
		user := c.Locals("user").(*jwt.Token)
		claims := user.Claims.(jwt.MapClaims)
		bind = fiber.Map{
			"username":       claims["username"].(string),
			"profilepicture": claims["profilepicture"].(string),
		}
		if view.NeedsQuerying {
			for _, variable := range view.QueryVariableNames {
				view.DatabaseQuery.Contains[variable] = c.Query(variable)
			}
			_, err := view.DatabaseQuery.GenerateQuery(DB)
			if err != nil {
				fmt.Println(err.Error())
			}
			for k := range view.DatabaseQuery.Result {
				for key, value := range view.DatabaseQuery.Result[k] {
					bind[key] = value
				}
			}
		}
		if err := c.Render(view.ViewPath, bind); err != nil {
			c.Status(500).Send(err.Error())
		}
	})
}

// DatabaseItemType Defines the allowed types of values for the database.
type DatabaseItemType string

func (itemType *DatabaseItemType) String() string {
	switch *itemType {
	case NULL:
		return "NULL"
	case INTEGER:
		return "INTEGER"
	case REAL:
		return "REAL"
	case TEXT:
		return "TEXT"
	case BLOB:
		return "BLOB"
	}
	return ""
}

const (
	// NULL Sqlite3. The value is a NULL value
	NULL DatabaseItemType = "NULL"
	// INTEGER Sqlite3. The value is a signed integer, stored in 1, 2, 3, 4, 6, or 8 bytes depending on the magnitude of the value.
	INTEGER DatabaseItemType = "INTEGER"
	// REAL Sqlite3. The value is a floating point value, stored as an 8-byte IEEE floating point number.
	REAL DatabaseItemType = "REAL"
	// TEXT Sqlite3. The value is a text string, stored using the database encoding (UTF-8, UTF-16BE or UTF-16LE).
	TEXT DatabaseItemType = "TEXT"
	// BLOB Sqlite3. The value is a blob of data, stored exactly as it was input.
	BLOB DatabaseItemType = "BLOB"
)

// DatabaseOperationType Defines the operation types for the database.
type DatabaseOperationType string

func (operationType *DatabaseOperationType) String() string {
	switch *operationType {
	case INSERT:
		return "INSERT"
	case SELECT:
		return "SELECT"
	case UPDATE:
		return "UPDATE"
	case DELETE:
		return "DELETE"
	}
	return ""
}

const (
	// INSERT sqlite3. SQLite INSERT INTO Statement is used to add new rows of data into a table in the database.
	INSERT DatabaseOperationType = "INSERT"
	// SELECT sqlite3. SQLite SELECT statement is used to fetch the data from a SQLite database table which returns data in the form of a result table.
	SELECT DatabaseOperationType = "SELECT"
	// UPDATE sqlite3. SQLite UPDATE statement is used to  modify the existing records in a table. You can use WHERE clause with UPDATE query to update selected rows, otherwise all the rows would be updated.
	UPDATE DatabaseOperationType = "UPDATE"
	// DELETE sqlite3. SQLite DELETE statement is used to delete  the existing records from a table. You can use WHERE clause with DELETE query to delete the selected rows, otherwise all the records would be deleted.
	DELETE DatabaseOperationType = "DELETE"
)

// DatabaseItems is a the item insertet into the database table.
// This is made so that a the database item name can be mapped to it's type.
type DatabaseItems map[string]DatabaseItemType

//DatabaseTable Descripes the structure of a database table.
type DatabaseTable struct {
	TableName string        `json:"TableName"`
	Items     DatabaseItems `json:"Items"`
}

// GenerateTable generates the given database table if it doesn't exist'.
func (database *DatabaseTable) GenerateTable(DB *sql.DB) error {
	// Setup the FileSettings table if it doesn't exist'
	Query := "CREATE TABLE IF NOT EXISTS " + database.TableName + "("
	Query += "ID INTEGER NOT NULL PRIMARY KEY AUTOINCREMENT"
	i := 0
	for key, sqlType := range database.Items {
		if i < len(database.Items) {
			Query += ", " + key + " " + sqlType.String()
		} else {
			Query += ", " + key + " " + sqlType.String()
		}
		i++
	}
	Query += ");"
	statement, err := DB.Prepare(Query)
	if err != nil {
		return err
	}
	statement.Exec()
	return nil
}

//DatabaseQuery the structure that containse teh data representation of a database query.
type DatabaseQuery struct {
	Result            []map[string]interface{}    `json:"Result"`
	VariableType      map[string]DatabaseItemType `json:"VariableType"`
	Contains          map[string]string           `json:"Contains"`
	Set               map[string]string           `json:"Set"`
	TableName         string                      `json:"TableName"`
	DatabaseOperation DatabaseOperationType       `json:"DatabaseOperation"`
}

// GenerateQuery constructs the query based on the information provided from the DatabaseQuery
func (query *DatabaseQuery) GenerateQuery(DB *sql.DB) (string, error) {
	Query := ""
	query.TypeCorretor()
	switch query.DatabaseOperation {
	case INSERT:
		Query = query.Insert()
	case SELECT:
		Query = query.Select()
	case UPDATE:
		Query = query.Update()
	case DELETE:
		Query = query.Delete()
	}
	rows, err := DB.Query(Query)
	if err != nil {
		return "", err
	}
	err = query.LoadResultIntoMap(rows)
	if err != nil {
		return "", err
	}
	// If you want it pretty-printed
	resultJSONBYTE, err := json.MarshalIndent(query.Result, "", "  ")
	if err != nil {
		return "", err
	}
	resultJSON := string(resultJSONBYTE)
	return resultJSON, nil
}

// TypeCorretor insures that items have the correct type.
func (query *DatabaseQuery) TypeCorretor() {
	for key, value := range query.Contains {
		if query.VariableType[key] == TEXT {
			query.Contains[key] = "\"" + value + "\""
		}
	}
	for key, value := range query.Set {
		if query.VariableType[key] == TEXT {
			query.Set[key] = "\"" + value + "\""
		}
	}
}

// Insert sqlite3. SQLite INSERT INTO Statement is used to add new rows of data into a table in the database.
func (query *DatabaseQuery) Insert() string {
	Query := query.DatabaseOperation.String() + " INTO " + query.TableName + " ("
	keyMap := make(map[int]string)
	i := 0
	for key := range query.Contains {
		keyMap[i] = key
		if i < len(query.Contains)-1 {
			Query += key + ","
		} else {
			Query += key + ")"
		}
		i++
	}
	Query += " VALUES ("
	for j := 0; j < i; j++ {
		if j < i-1 {
			Query += query.Contains[keyMap[j]] + ","
		} else {
			Query += query.Contains[keyMap[j]] + ")"
		}
	}
	return Query
}

// Select sqlite3. SQLite SELECT statement is used to fetch the data from a SQLite database table which returns data in the form of a result table.
func (query *DatabaseQuery) Select() string {
	Query := query.DatabaseOperation.String() + " * FROM " + query.TableName
	i := 0
	if len(query.Contains) > 0 {
		Query += " WHERE "
		for key, value := range query.Contains {
			if i < len(query.Contains)-1 {
				Query += key + "=" + value + " AND "
			} else {
				Query += key + "=" + value
			}
			i++
		}
	}
	return Query
}

// Update sqlite3. SQLite UPDATE statement is used to  modify the existing records in a table. You can use WHERE clause with UPDATE query to update selected rows, otherwise all the rows would be updated.
func (query *DatabaseQuery) Update() string {
	Query := query.DatabaseOperation.String() + " " + query.TableName + " SET "
	i := 0
	for key, value := range query.Set {
		if i < len(query.Set)-1 {
			Query += key + "=" + value + " AND "
		} else {
			Query += key + "=" + value
		}
		i++
	}
	Query += " WHERE "
	i = 0
	for key, value := range query.Contains {
		if i < len(query.Contains)-1 {
			Query += key + "=" + value + " AND "
		} else {
			Query += key + "=" + value
		}
		i++
	}
	return Query
}

// Delete sqlite3. SQLite DELETE statement is used to delete  the existing records from a table. You can use WHERE clause with DELETE query to delete the selected rows, otherwise all the records would be deleted.
func (query *DatabaseQuery) Delete() string {
	Query := query.DatabaseOperation.String() + " FROM " + query.TableName + " WHERE "
	i := 0
	for key, value := range query.Contains {
		if i < len(query.Contains)-1 {
			Query += key + "=" + value + " AND "
		} else {
			Query += key + "=" + value
		}
		i++
	}
	return Query
}

// LoadResultIntoMap Loads the result of a database query into a map.
func (query *DatabaseQuery) LoadResultIntoMap(rows *sql.Rows) error {
	var columns []string
	columns, err := rows.Columns()
	if err != nil {
		return err
	}

	colNum := len(columns)

	for rows.Next() {
		// Prepare to read row using Scan
		r := make([]interface{}, colNum)
		for i := range r {
			r[i] = &r[i]
		}

		// Read rows using Scan
		err = rows.Scan(r...)
		if err != nil {
			return err
		}

		// Create a row map to store row's data
		var row = map[string]interface{}{}
		for i := range r {
			row[columns[i]] = r[i]
		}

		// Append to the final results slice
		query.Result = append(query.Result, row)
	}
	return nil
}
