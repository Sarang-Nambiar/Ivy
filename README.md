# Ivy

## Understanding the implementation flow:
1. The core implementation of Integrated shared Virtual memory at Yale has more or less remained the same. The only changes made are the way the Read and Write requests are handled when the server has no records of previous requests:
    - For Read, if the records stored at the central manager server are empty, then the server returns a Page not found error as shown below:
    ![Screenshot 2024-12-05 202115](https://github.com/user-attachments/assets/bfd673eb-852b-491d-a67a-3637d0a5a056)
    - For Write, if the records stored at the central manager server are empty, then the server creates a new record for requested page and grants the client Write permission for that page.
2. For the Backup flow, the primary central manager sends backup messages over to the backup central manager every 5 seconds(configurable). The backup central manager updates its metadata and does a health check on the primary central manager every 5 seconds(configurable). If the primary central manager is down, the backup central manager takes over as the primary central manager. If the backup central manager detects that the primary central manager is back alive, it returns control over to the primary central manager.

## How to run the code:
1. First open 12 powershell terminals(1 primary, 1 backup CM and 10 clients) and make sure you are in this project root directory. 
2. There are three main components to the Ivy system and each of them have their own run command:
    - Primary Central Manager Server
    ```powershell
    ivy.exe -cm
    ```
    - Client
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

![Screenshot 2024-12-12 164507](https://github.com/user-attachments/assets/1c2c78f6-67a8-4a10-9cac-7116e5cd07be)

## Things to consider:
1. Currently, the client can request in either a completely randomized read/write request or a percentage based read/write request. The percentage based read/write request can be configured in client.go by providing the percentage of READ requests to be made by the client and the rest will be WRITE requests. You can choose which type of request to make by commenting out the other type of request in the client.go file as shown:

![Screenshot 2024-12-11 213303](https://github.com/user-attachments/assets/6c506e2f-b10b-44f0-aebd-87e7d5e8ab13)

2. The read/write requests end after 10 requests of either type have been successfully completed. After all the requests from all the nodes are completed, the central manager which is alive will print out the average time taken for each type of request as shown below:

![image](https://github.com/user-attachments/assets/4ee7adac-b642-4221-86f1-40c8fa787a2b)

![Screenshot 2024-12-11 213242](https://github.com/user-attachments/assets/ad429482-8184-4e07-aecd-9094d57750bd)

3. Time taken to get the page with required permission from the cache isn't taken into account in the average time calculation.

## Is the current Fault tolerant implementation of Ivy sequentially consistent?
To maintain sequential consistency, two conditions must be met:
1. All operations in one machine are executed in order.
2. All machine observe results according to some total ordering.

Condition 1 is met to by default since it is a programming language. For condition 2, the writes from all the clients are appended to a queue based on whatever request arrived first at the central manager. The next write operation is not executed until the first write operation in the queue is completed. This ensures some total ordering to the write operations. When the primary central manager goes down, the backup central manager takes over and continues to maintain the total ordering from the backed up metadata. Hence, the current fault tolerant implementation of Ivy is sequentially consistent.

## Scenario 1: Without any faults, Comparison of the performance of Ivy with and without the backup central manager with randomized read and write requests.

Since there are no faults, the performance of Ivy with and without the backup central manager is around the same for both cases:

Basic Ivy:
| Request Type | Highest     | Lowest     | Average    |
|--------------|-------------|------------|------------|
| READ         | 5.6587ms    | 2.7883ms   | 4.7350ms   |
| WRITE        | 32.9928ms   | 2.7268ms   | 14.2624ms  |

Fault Tolerant Ivy:
| Request Type | Highest     | Lowest     | Average    |
|--------------|-------------|------------|------------|
| READ         | 4.5815ms    | 2.792ms    | 8.6842ms   |
| WRITE        | 33.3176ms   | 3.9434ms   | 16.7383ms  |

## Scenario 2: Without any faults, Comparison of the performance of Ivy with and without the backup central manager for read-intensive and write-intensive workloads.

### Basic Ivy:

Read Intensive(90% read, 10% write):
| Request Type | Average   |
|--------------|-----------|
| READ         | 4.8558ms  |
| WRITE        | 10.5462ms |

Write Intensive(10% read, 90% write):
| Request Type | Average   |
|--------------|-----------|
| READ         | 4.6782ms  |
| WRITE        | 13.7364ms |

### Fault Tolerant Ivy:

Read Intensive(90% read, 10% write):
| Request Type | Average   |
|--------------|-----------|
| READ         | 5.5431ms  |
| WRITE        | 13.2370ms |

Write Intensive(10% read, 90% write):
| Request Type | Average   |
|--------------|-----------|
| READ         | 4.8347ms  |
| WRITE        | 14.0659ms |

## Scenario 3: In the presence of a single fault, Primary CM goes down or restarts randomly.

### Primary CM goes down:

For Randomized Read/Write:
| Request Type | Average   |
|--------------|-----------|
| READ         | 4.6193ms  |
| WRITE        | 12.3003ms |

For Read Intensive(90% read, 10% write):
| Request Type | Average   |
|--------------|-----------|
| READ         | 4.1269ms  |
| WRITE        | 4.9472ms  |

For Write Intensive(10% read, 90% write):
| Request Type | Average   |
|--------------|-----------|
| READ         | 4.2766ms  |
| WRITE        | 14.8333ms |

### Primary CM restarts:

For Randomized Read/Write:
| Request Type | Average   |
|--------------|-----------|
| READ         | 4.5351ms  |
| WRITE        | 9.4223ms  |

For Read Intensive(90% read, 10% write):
| Request Type | Average      |
|--------------|--------------|
| READ         | 4.3412ms  |
| WRITE        | 7.9190ms  |

For Write Intensive(10% read, 90% write):
| Request Type | Average   |
|--------------|-----------|
| READ         | 5.9715ms  |
| WRITE        | 12.0355ms |

## Scenario 4: In the presence of multiple faults, Primary CM goes down and restarts randomly multiple times.

For Randomized Read/Write:
| Request Type | Average   |
|--------------|-----------|
| READ         | 4.3343ms  |
| WRITE        | 9.8712ms  |

For Read Intensive(90% read, 10% write):
| Request Type | Average   |
|--------------|-----------|
| READ         | 4.2669ms  |
| WRITE        | 8.1538ms  |

For Write Intensive(10% read, 90% write):
| Request Type | Average   |
|--------------|-----------|
| READ         | 6.2179ms  |
| WRITE        | 14.0183ms |

## Scenario 5: In the presence of multiple faults, Primary CM and backup CM goes down and restarts randomly multiple times.

For Randomized Read/Write:
| Request Type | Average   |
|--------------|-----------|
| READ         | 4.5989ms  |
| WRITE        | 11.2581ms |

For Read Intensive(90% read, 10% write):
| Request Type | Average   |
|--------------|-----------|
| READ         | 4.5716ms  |
| WRITE        | 9.8187ms  |

For Write Intensive(10% read, 90% write):
| Request Type | Average   |
|--------------|-----------|
| READ         | 5.4770ms  |
| WRITE        | 13.6728ms |

## Conclusion:
The performance of Ivy with or without the backup central manager is around the same even in the presence of faults for all the scenarios mentioned as the metadata is always synchronized between the primary and backup central managers. The only delay that could occur is when the central manager declaring that it is taking over as the "primary" central manager.
