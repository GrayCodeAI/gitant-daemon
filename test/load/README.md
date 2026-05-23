# Load Tests

Run with k6: https://k6.io

## API Load Test
```bash
k6 run -e GITANT_URL=http://localhost:7777 api-test.js
```

## With authentication
```bash
k6 run -e GITANT_URL=http://localhost:7777 -e GITANT_TOKEN=<your-token> api-test.js
```

## Git Protocol Load Test
```bash
k6 run -e GITANT_URL=http://localhost:7777 git-protocol-test.js
```
