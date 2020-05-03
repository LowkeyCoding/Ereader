const credentialsForm = document.getElementById("credentialsForm");
const actionButton = document.getElementById("actionButton");
const signinButton = document.getElementById("signinButton");
const signupButton = document.getElementById("signupButton");

signinButton.addEventListener("click",()=>{
    console.log("signinButton classname: ", signinButton.className)
    signinButton.className = "active button"
    signupButton.className = "inactive underlineHover button"
    actionButton.value = "Sign In"
    credentialsForm.action = "/signin"
})
signupButton.addEventListener("click",()=>{
    console.log("signupButton classname: ", signupButton.className);
    signupButton.className = "active button"
    signinButton.className = "inactive underlineHover button"
    actionButton.value = "Sign Up"
    credentialsForm.action = "/signup"
})