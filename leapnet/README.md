##package leapnet

leapnet implements various strategies for assembling a leaps service, the classic example is a single node solution where leaps acts as a single http endpoint. However, leaps is made up of modular components with an aim to make them easily distributed for both redundancy and load balancing, the plan is to have multiple leapnet configurations catered towards linking these components together.

STATUS: INCOMPLETE

TODO:
- Leaps websocket client for bridging connection between curators.
- HTTP poller client
