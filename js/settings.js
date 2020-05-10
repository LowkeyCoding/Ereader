function post(path, params, callback) {
    let formData = new FormData();
    for (const key in params) {
      if (params.hasOwnProperty(key)) {
        formData.append(key, params[key]);
      }
    }
    fetch(path, {
      method: 'POST',
      body: formData 
    }).then(response => {
        callback(response);
    }).catch(err => console.log(err));
}
let button = document.getElementById("actionButton");
let Extension = document.getElementById("Extension");
let ApplicationLink = document.getElementById("ApplicationLink");
button.onclick = () => {
    let params = {
        "Result": null,
        "VariableType": '{"Extension": "TEXT","Username": "TEXT"}',
        "Contains": '{"Extension": "'+Extension.value+'","Username": "'+username+'"}',
        "Set": null,
        "TableName": "FileSettings",
        "DatabaseOperation": "SELECT"
    }
    post("/query", params, (response) => {
        console.log(response);
        response.json().then(data => {
            if (data) {
                response = data[0]
            } else {
                response.ID = null
            }
            if (response.ID != null) {
                let params = {
                "Result": null,
                "VariableType": '{"Extension": "TEXT", "ApplicationLink": "TEXT", "Username": "TEXT"}',
                "Contains": '{"Extension": "'+Extension.value+'", "Username": "'+username+'"}',
                "Set": '{"ApplicationLink": "'+ApplicationLink.value+'"}',
                "TableName": "FileSettings",
                "DatabaseOperation": "UPDATE"
                }
                post("/query", params, (response) => {alert(response.statusText)})
            } else {
                let params = {
                    "Result": null,
                    "VariableType": '{"Extension": "TEXT", "ApplicationLink": "TEXT", "Icon": "TEXT", "Username": "TEXT"}',
                    "Contains": '{"Extension": "'+Extension.value+'", "ApplicationLink": "'+ApplicationLink.value+'", "Icon": "'+Extension.value.substr(1)+'", "Username": "'+username+'"}',
                    "Set": null,
                    "TableName": "FileSettings",
                    "DatabaseOperation": "INSERT"
                }
                post("/query", params, (response) => {alert(response.statusText)})
            }
        })
    })
}