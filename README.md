# Ivy

## Understanding the implementation flow:
1. The core implementation of Integrated shared Virtual memory at Yale has more or less remained the same. The only changes made are the way the Read and Write requests are handled when the server has no records of previous requests:
    - For Read, if the records stored at the central manager server are empty, then the server returns a Page not found error as shown below:
    <!-- put screenshot of the error here -->
    - For Write, if the records stored at the central manager server are empty, then the server creates a new record for requested page and grants the client Write permission for that page.
2. For the Backup flow, the primary central manager sends backup messages over to the backup central manager every 5 seconds(configurable). The backup central manager updates its metadata and does a health check on the primary central manager every 5 seconds(configurable). If the primary central manager is down, the backup central manager takes over as the primary central manager. If the backup central manager detects that the primary central manager is back alive, it returns control over to the primary central manager.

## How to run the code:
1. First open 12 powershell terminals(1 primary, 1 backup CM and 10 clients) and make sure you are in this project root directory. 
2. There are three main components to the Ivy system and each of them have their own run command:
    - Primary Central Manager Server
    ```powershell
    ivy.exe -cm
    ```
    - Client - -cl
    ```powershell
    ivy.exe -cl
    ```
    - Backup Central Manager Server
    ```powershell
    ivy.exe -b
    ```
    The ideal order to run the components is to first run the primary central manager server, then the backup central manager server and then the clients. The clients will automatically connect to the primary central manager server and the backup central manager server will automatically connect to the primary central manager server start its backup process after starting the Read and Write requests of the clients.

## How to read the output:
The terminal output for all the nodes in the network follow a similar format:
[Node Type] [Node ID(if client)] [Event]

<!-- show sample output here -->

## Is the current Fault tolerant implementation of Ivy sequentially consistent?
To maintain sequential consistency, two conditions must be met:
1. All operations in one machine are executed in order.
2. All machine observe results according to some total ordering.

Condition 1 is met to by default since it is a programming language. For condition 2, the writes from all the clients are appended to a queue based on whatever request arrived first at the central manager. The next write operation is not executed until the first write operation in the queue is completed. This ensures some total ordering to the write operations. When the primary central manager goes down, the backup central manager takes over and continues to maintain the total ordering from the backed up metadata. Hence, the current fault tolerant implementation of Ivy is sequentially consistent.

