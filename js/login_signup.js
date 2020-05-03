const credentialsForm = document.getElementById("credentialsForm");
const actionButton = document.getElementById("actionButton");
const signinButton = document.getElementById("signinButton");
const signupButton = document.getElementById("signupButton");
const forgotPasswordButton = document.getElementById("forgotPassword");
var profilePictureInput = document.getElementById("profilepicture");
const icon = document.getElementById("icon");

signinButton.addEventListener("click", () => {
    signinButton.className = "active button";
    signupButton.className = "inactive underlineHover button";
    credentialsForm.removeChild(profilePictureInput);
    profilePictureInput = ""
    actionButton.value = "Sign In";
    credentialsForm.action = "/signin";
})

signupButton.addEventListener("click", () => {
    signupButton.className = "active button";
    signinButton.className = "inactive underlineHover button";
    if(!profilePictureInput){
        profilePictureElement = document.createElement("input");
        profilePictureElement.type = "text";
        profilePictureElement.id = "profilepicture";
        profilePictureElement.name = "profilepicture";
        profilePictureElement.placeholder = "profilepicture";
        credentialsForm.insertBefore(profilePictureElement,actionButton);
        profilePictureInput = document.getElementById("profilepicture");
        profilePictureInput.onchange = changeProfilePicture
    }
    actionButton.value = "Sign Up";
    credentialsForm.action = "/signup";
})

forgotPasswordButton.addEventListener("click", () => {
    alert("To bad it takes to long to implement that feature.");
    alert("Just manually hash your new password and insert it into the database like chad");
})
const changeProfilePicture =  () => {
    icon.src = profilePictureInput.value ||"./media/icons/user-circle.svg";
    icon.onerror = () => {
        icon.src = "./media/icons/user-circle.svg"
        alert("Image couldn't be loaded please check the link and try again'")
    }
}
profilePictureInput.onchange = changeProfilePicture