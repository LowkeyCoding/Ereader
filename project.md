# Libary Management Softwares
Problemet jeg har fundet er at der ikke er et godt program til at læse bøger i  
der kan hold styr på hvor langt man er der virker på alle platformer.  
Derfor har jeg tænkt at lave et web baset biblotekts håndternings system med en backend i golang.  
Til et ordent ligt system har jeg disse krav:  
* PDF support.
* Et system til at upload og katagorisere bøger.
* Et system til at huske hvor langt man er i en bog uafhængig af skærm størrelsen.
* En frontend der gør det nemt at vælge og søge efter bøger.

Udover det har jeg nogle features der kunne være delige at have

* Automatisk downloadedt meta data. 
* Download af bøger udfra en forspørgelse.
* Automatisk fil konversion udfra MIME-TYPE.
* Support for flere.
* Implementation af ETAGS og GZIP til at optimere mængden af data brugt.
* Support til at have flere brugere med hver deres.
* Support for flere fil formater.