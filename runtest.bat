@mkdir bin

8g -o bin\markup.8 html.go markup.go
@if ERRORLEVEL 1 EXIT /B 1

8g -o bin\upskirt_ref_test.8 -I bin upskirt_ref_test.go 
@if ERRORLEVEL 1 EXIT /B 1

8l -o bin\upskirtreftest.exe -L bin bin\upskirt_ref_test.8
@if ERRORLEVEL 1 EXIT /B 1

bin\upskirtreftest.exe
