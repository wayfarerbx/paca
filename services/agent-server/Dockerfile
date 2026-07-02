# Pre-installs the Paca MCP server globally so `npx -y @paca-ai/paca-mcp`
# resolves from the local install instead of downloading the package on
# every ephemeral sandbox container boot.
FROM ghcr.io/openhands/agent-server:latest-python

# The base image runs as the unprivileged `openhands` user, which doesn't own
# npm's global install dir — switch to root for the install, then drop back
# so the container still runs unprivileged at runtime.
USER root
RUN npm install -g @paca-ai/paca-mcp
USER openhands
