curl -s -X POST -H "Content-Type: application/json" -d '{"parts":[{"type":"text","text":"write a long story about a brave knight"}]}' http://localhost:4116/session/ses_2e2b09429ffeRjjp3BzvRnNA5U/prompt &
sleep 2
curl -s -X POST -w "%{http_code}" http://localhost:4116/session/ses_2e2b09429ffeRjjp3BzvRnNA5U/abort
