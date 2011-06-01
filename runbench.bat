@mkdir bin

8g -o bin\markup.8 html.go markup.go
@if ERRORLEVEL 1 EXIT /B 1

8g -o bin\upskirt_ref_test.8 -I bin upskirt_ref_test.go 
@if ERRORLEVEL 1 EXIT /B 1

8g -o bin\bench.8 -I bin bench.go
@if ERRORLEVEL 1 EXIT /B 1

8l -o bin\bench.exe -L bin bin\bench.8
@if ERRORLEVEL 1 EXIT /B 1

bin\bench.exe
