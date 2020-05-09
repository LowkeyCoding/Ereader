package extension

import (
	"database/sql"
	"encoding/json"
	"fmt"
	"io/ioutil"

	"github.com/dgrijalva/jwt-go"
	"github.com/gofiber/fiber"
)

// < ----- Extension Generators ----- >

//Extension descripes the structure of an extension.
type Extension struct {
	Name           string          `json:"Name"`
	Path           string          `json:"Path"`
	Views          []View          `json:"Views"`
	DatabaseTables []DatabaseTable `json:"DatabaseTable"`
}

// LoadExtension loads the extension from the config file.
func (extension *Extension) LoadExtension() {
	jsonMap := make(map[string]interface{})
	config, err := ioutil.ReadFile(extension.Path + "/config.json")
	if err != nil {
		fmt.Println("Invalid config: ", err.Error())
	}
	err = json.Unmarshal(config, &jsonMap)
	if err != nil {
		fmt.Println("Invalid config: ", err.Error())
	}
	Extension := extension.InterfaceToExtension("", jsonMap)
	extension.Name = Extension.Name
	extension.Views = Extension.Views
	extension.DatabaseTables = Extension.DatabaseTables
}

// GenerateStaticPaths generates paths for the css and js files to be served.
func (extension Extension) GenerateStaticPaths(app *fiber.App) {
	app.Static(extension.Name+"/css", extension.Path+"/css")
	app.Static(extension.Name+"/js", extension.Path+"/js")
}

// Setup generates the database tables and views.
func (extension *Extension) Setup(app *fiber.App, DB *sql.DB) error {
	extension.GenerateStaticPaths(app)
	for _, databaseTable := range extension.DatabaseTables {
		databaseTable.GenerateTable(DB)
	}
	for _, view := range extension.Views {
		view.GenerateView(app, DB)
	}
	return nil
}

// InterfaceToExtension converts an interface to the Extension structure.
func (extension *Extension) InterfaceToExtension(space string, m map[string]interface{}) Extension {
	var Extension Extension
	for k, v := range m {
		if k == "Name" {
			Extension.Name = v.(string)
		} else if k == "Views" {
			var Views []View
			if mv, ok := v.(map[string]interface{}); ok {
				for _, v := range mv {
					Views = append(Views, extension.InterfaceToView(v.(map[string]interface{})))
				}
			}
			Extension.Views = Views
		} else if k == "DatabaseTables" {
			var DatabaseTables []DatabaseTable
			if mv, ok := v.(map[string]interface{}); ok {
				for _, v := range mv {
					DatabaseTables = append(DatabaseTables, extension.InterfaceToTable(v.(map[string]interface{})))
				}
				Extension.DatabaseTables = DatabaseTables
			}
		}
	}
	return Extension
}

// InterfaceToView converts an interface to the View structure.
func (extension *Extension) InterfaceToView(m map[string]interface{}) View {
	var View View
	for k, v := range m {
		switch k {
		case "Path":
			View.Path = v.(string)
		case "ViewPath":
			View.ViewPath = extension.Path + v.(string)
		case "NeedsQuerying":
			View.NeedsQuerying = v.(bool)
		case "QueryVariableNames":
			var QueryVariableNames []string
			for _, vv := range v.([]interface{}) {
				QueryVariableNames = append(QueryVariableNames, vv.(string))
			}
			View.QueryVariableNames = QueryVariableNames
		case "DatabaseQuery":
			if mv, ok := v.(map[string]interface{}); ok {
				View.DatabaseQuery = extension.InterfaceToQuery(mv)
			}
		}
	}
	return View
}

// InterfaceToQuery converts an interface to the DatabaseQuery structure.
func (extension *Extension) InterfaceToQuery(m map[string]interface{}) DatabaseQuery {
	var Query DatabaseQuery
	for k, v := range m {
		switch k {
		case "VariableType":
			VariableType := make(map[string]DatabaseItemType)
			for kk, vv := range v.(map[string]interface{}) {
				VariableType[kk] = DatabaseItemType(vv.(string))
			}
			Query.VariableType = VariableType
		case "Contains":
			//Query.Contains = v.(map[string]interface{})
			Contains := make(map[string]string)
			for kk, vv := range v.(map[string]interface{}) {
				Contains[kk] = vv.(string)
			}
			Query.Contains = Contains

		case "Set":
			Set := make(map[string]string)
			for kk, vv := range v.(map[string]interface{}) {
				Set[kk] = vv.(string)
			}
			Query.Set = Set
		case "TableName":
			Query.TableName = v.(string)
		case "DatabaseOperation":
			Query.DatabaseOperation = DatabaseOperationType(v.(string))
		}
	}
	return Query
}

// InterfaceToTable converts an interface to the DatabaseTable structure.
func (extension *Extension) InterfaceToTable(m map[string]interface{}) DatabaseTable {
	var DatabaseTable DatabaseTable
	for k, v := range m {
		switch k {
		case "TableName":
			DatabaseTable.TableName = v.(string)
		case "Items":
			Items := make(map[string]DatabaseItemType)
			for kk, vv := range v.(map[string]interface{}) {
				Items[kk] = DatabaseItemType(vv.(string))
			}
			DatabaseTable.Items = Items
		}
	}
	return DatabaseTable
}

// Extensions is a array of containing multiple instances of Extension.
type Extensions []Extension

// LoadExtensions loads all extensions from the extensions folder.
func (extensions *Extensions) LoadExtensions(app *fiber.App, DB *sql.DB) {
	files, err := ioutil.ReadDir("./Extensions")
	if err != nil {
		fmt.Println(err.Error())
	}

	for _, file := range files {
		if file.IsDir() {
			Extension := Extension{Path: "./Extensions/" + file.Name()}
			Extension.LoadExtension()
			Extension.Setup(app, DB)
			*extensions = append(*extensions, Extension)
		}
	}
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
