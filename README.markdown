# Ereader
# /home
the home route taks in a path, which is used to show the folder in the given directory. 

    ?path=<path>

![alt text](/media/screenshots/Home_4.png "Home_4")
![alt text](/media/screenshots/Home_5.png "Home_5")
# /login
![alt text](/media/screenshots/Signin.png "Signin")
![alt text](/media/screenshots/Signup_1.png "Signup 1")
![alt text](/media/screenshots/Signup_2.png "Signup 2")
# /pdf
The /pdf route takes in the path of the file, the hash of the file, and the usernames of the current user to dispaly the selectet pdf if it exists in the database. If it doesn't exist it will be created.

    ?Path=<Path>&Hash=<Hash>&Username=<Username>
This route has been generated from the PDFReader extension
[alt text](/media/screenshots/PDF_1.png "PDF 1")
## Pdf reader usage
Use the right arrow key to move forwards a page.  
Use the left arrow key to move back a page.  
Hold escape to go back to the /home in the same path as the one you used to open the file.  
