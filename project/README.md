# How to run tests

## Locally

There is a Taskfile in the project root directory. 
You need to run before the test
* gateway container
* redis container
* service container or service locally


Before running the tests, you need to adjust the environment variables in the `test-component-local` task.

```bash
task local-test-deps # run gateway and redis
task app # run service
task test-component-local # run tests
```


## Diagram



```mermaid
sequenceDiagram
    participant C as Client
    participant S as Our Service
    participant DB as Database
    participant MB as Message Broker
    participant TP as 3rd Party (DeadNation)
    participant Sp as Spreadsheets
    participant F as File Storage
    participant R as Receipts Client

    C->>+S: POST /book-tickets

    rect rgba(173, 216, 230, 0.2)
        Note over S,DB: Single Transaction
        S->>DB: Check show & tickets availability
        S->>DB: Create booking
        S->>DB: Add BookingMade to outbox
    end

    S-->>-C: Response

    rect rgba(230, 230, 230, 0.3)
        Note over DB,TP: ticket_booking_handler
        DB->>S: Deliver BookingMade to ticket_booking_handler
        S->>TP: DeadNation.BookTickets
    end

    Note over TP,S: Async callback (после обработки на стороне DeadNation)
    TP-->>S: POST /tickets-status

    alt Tickets Confirmed
        S->>MB: Publish TicketBookingConfirmed

        par Process Confirmed Tickets
            Note over MB,Sp: ticket_to_print_handler
            MB->>S: Deliver TicketBookingConfirmed to ticket_to_print_handler
            S->>Sp: AppendRow("tickets-to-print")

            Note over MB,F: prepare_tickets_handler
            MB->>S: Deliver TicketBookingConfirmed to prepare_tickets_handler
            S->>F: Upload(generatedTicket)
            S->>MB: Publish TicketPrinted

            Note over MB,R: issue_receipt_handler
            MB->>S: Deliver TicketBookingConfirmed to issue_receipt_handler
            S->>R: IssueReceipt
            S->>MB: Publish TicketReceiptIssued

            Note over MB,DB: store_tickets_handler
            MB->>S: Deliver TicketBookingConfirmed to store_tickets_handler
            S->>DB: ticketsRepository.Create
        end

    else Tickets Cancelled
        S->>MB: Publish TicketBookingCanceled

        par Process Cancelled Tickets
            Note over MB,Sp: refund_ticket_handler
            MB->>S: Deliver TicketBookingCanceled to refund_ticket_handler
            S->>Sp: AppendRow("tickets-to-refund")

            Note over MB,DB: remove_tickets_handler
            MB->>S: Deliver TicketBookingCanceled to remove_tickets_handler
            S->>DB: ticketsRepository.Delete
        end
    end
```


### Events 
```mermaid
graph LR
   A[Event Bus] --> B['events' topic]
   B --> C[Data lake consumer group]
   B --> D[Event forwarder consumer group]
   C --> E[Data lake]
   D --> F['events.BookingMade' topic]
   D --> G['events.TicketBookingConfirmed' topic]
   D --> H['events.TicketReceiptIssued' topic]
   D --> I['events.TicketPrinted' topic]
   D --> J['events.TicketRefunded' topic]
   D --> K['events.ReadModelIn' topic]
```

### Internal Events
```mermaid
graph LR
    A[Event Bus] --> B['events' topic]
    B --> C[Data lake consumer]
    B --> D[Events forwarder]
    C --> E[Data lake]
    D --> F['events.BookingMade' topic]
    D --> G['events.TicketBookingConfirmed' topic]
    D --> H['events.TicketReceiptIssued' topic]
    D --> I['events.TicketPrinted' topic]
    D --> J['events.TicketRefunded' topic]
    D --> K['events.ReadModelIn' topic]
    A -- publish directly --> L['internal-events.svc-tickets.InternalOpsReadModelUpdated'<br>topic]


classDef orange fill:#f96,stroke:#333,stroke-width:4px;
class L orange
```


## Tracing with OpenTelemetry and Jaeger

![img.png](img.png)