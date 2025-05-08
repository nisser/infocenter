Notes:</br>
    Client timeout is set to 1 minute.</br>
    Timeout time can be changed by editing the global TIMEOUT variable.

Flags:</br>
    Enable debugging: --debug=true/false (false by default).</br>
    Enable custom host url: --url=http://anyurl:1337/infocenter (http://localhost:8080/infocenter by default).

Usage:</br>
    1. Open two terminals.</br>
    2. In terminal #1 start the program by using "go run .". Add flags if needed (see: Flags).</br>
    3. Open a browser window, Google Chrome is recommended.</br>
    4. In the browser, open the link you specified with the flag. If not specified, use the default "http://localhost:8080/infocenter/test". Remember to add the topic at the end of the url when connecting.</br>
    5. If everything is done correctly, you should see "Client initialized" in the browser.</br>
    6. Try sending a POST request to the topic using terminal #2 (http POST http://localhost:8080/infocenter/test --raw "hello").</br>
    7. Message will show up in all browsers connected to the topic.</br>
    8. Do nothing for a minute (by default) and a timeout event will fire, disconnecting the client from the topic.</br>
