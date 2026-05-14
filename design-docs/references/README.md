# Design References

This directory contains reference materials for system design and implementation.

## External References

| Name                              | URL                                                                                                         | Description                                                                               |
| --------------------------------- | ----------------------------------------------------------------------------------------------------------- | ----------------------------------------------------------------------------------------- |
| TypeScript Documentation          | https://www.typescriptlang.org/docs/                                                                        | Official TypeScript documentation                                                         |
| Bun Documentation                 | https://bun.sh/docs                                                                                         | Official Bun runtime documentation                                                        |
| Amazon S3 Event Notifications     | https://docs.aws.amazon.com/AmazonS3/latest/userguide/notification-how-to-event-types-and-destinations.html | Official Amazon S3 event notification types and destination documentation                 |
| Amazon S3 EventBridge Integration | https://docs.aws.amazon.com/AmazonS3/latest/userguide/EventBridge.html                                      | Official Amazon S3 integration documentation for EventBridge event delivery               |
| Amazon S3 Event Message Structure | https://docs.aws.amazon.com/AmazonS3/latest/userguide/notification-content-structure.html                   | Official Amazon S3 event notification payload structure                                   |
| Vercel Chat SDK Documentation     | https://chat-sdk.dev/docs                                                                                   | Official Chat SDK documentation for multi-platform chat bot adapters and event handling   |
| Vercel Chat SDK Platform Adapters | https://chat-sdk.dev/docs/adapters                                                                          | Official adapter capability matrix for Slack, Discord, Telegram, and other chat providers |
| Vercel AI Elements                | https://vercel.com/changelog/introducing-ai-elements                                                        | Official Vercel AI Elements announcement and integration notes with the Vercel AI SDK     |

## Reference Documents

Reference documents should be organized by topic:

```
references/
├── README.md              # This index file
├── typescript/            # TypeScript patterns and practices
└── <topic>/               # Other topic-specific references
```

## Adding References

When adding new reference materials:

1. Create a topic directory if it does not exist
2. Add reference documents with clear naming
3. Update this README.md with the reference entry
