![](.media/logo.png)

# Build
`make build`

# Compress (optional, `upx` installed required)
`make build && make compress`

# Files structure
```
.
├── autodeploy.run
├── config
│   ├── config.yml
│   └── one.sh
└── logs
```

# URLs
- Run autodeploy: `/run/[serviceId]`
- Get last logs: `/logs/[logsKey]`
