import type { OpenClawPluginApi } from "openclaw/plugin-sdk";
import { Client } from "@modelcontextprotocol/sdk/client/index.js";
import { StdioClientTransport } from "@modelcontextprotocol/sdk/client/stdio.js";
import { Type } from "@sinclair/typebox";
import { spawn } from "child_process";
import { resolve } from "path";
import { homedir } from "os";

let client: Client | null = null;
let transport: StdioClientTransport | null = null;

function json(data: unknown) {
  return {
    content: [{ type: "text" as const, text: JSON.stringify(data, null, 2) }],
    details: data,
  };
}

const SWAT_BINARY = resolve(homedir(), ".local", "bin", "swat");

const TOOLS = [
  {
    name: "swat_dispatch",
    label: "SWAT Dispatch",
    description: "Dispatch a new task to a SWAT squad. Squad is auto-classified. Returns immediately; task runs in background.",
    parameters: Type.Object({
      brief: Type.String({ description: "Task description" }),
      details: Type.Optional(Type.String({ description: "Additional details" })),
    }),
  },
  {
    name: "swat_ops",
    label: "SWAT Operations",
    description: "List SWAT operations with optional filters. Returns counts and matching operations.",
    parameters: Type.Object({
      status: Type.Optional(Type.String({ description: "Filter by status (queued/active/completed/failed)" })),
      since: Type.Optional(Type.String({ description: "Only return terminal ops after this RFC3339 timestamp" })),
      limit: Type.Optional(Type.Number({ description: "Max results to return (default 50)" })),
      offset: Type.Optional(Type.Number({ description: "Skip first N results (default 0)" })),
    }),
  },
  {
    name: "swat_cancel",
    label: "SWAT Cancel",
    description: "Cancel a SWAT operation",
    parameters: Type.Object({
      operation_id: Type.String({ description: "Operation ID to cancel" }),
    }),
  },
  {
    name: "swat_squads",
    label: "SWAT Squads",
    description: "List installed SWAT squads",
    parameters: Type.Object({}),
  },
  {
    name: "swat_schedule",
    label: "SWAT Schedule",
    description: "Manage scheduled recurring tasks (create/list/delete). Zero LLM cost.",
    parameters: Type.Object({
      action: Type.String({ description: "Action: create, list, delete" }),
      brief: Type.Optional(Type.String({ description: "Task description (create)" })),
      cron: Type.Optional(Type.String({ description: "Cron expression, 5-field: min hour dom month dow (create)" })),
      details: Type.Optional(Type.String({ description: "Additional details (create)" })),
      timezone: Type.Optional(Type.String({ description: "IANA timezone, e.g. Asia/Shanghai (default: UTC)" })),
      name: Type.Optional(Type.String({ description: "Human-readable name" })),
      id: Type.Optional(Type.String({ description: "Schedule ID (delete)" })),
    }),
  },
  {
    name: "swat_squad_browse",
    label: "SWAT Squad Browse",
    description: "List all squads available in the marketplace",
    parameters: Type.Object({}),
  },
  {
    name: "swat_squad_install",
    label: "SWAT Squad Install",
    description: "Install a squad from the marketplace",
    parameters: Type.Object({
      squad: Type.String({ description: "Squad name to install" }),
    }),
  },
  {
    name: "swat_squad_uninstall",
    label: "SWAT Squad Uninstall",
    description: "Uninstall a squad and clean up orphaned dependencies",
    parameters: Type.Object({
      squad: Type.String({ description: "Squad name to uninstall" }),
      purge: Type.Optional(Type.Boolean({ description: "Also delete runtime data (default: false)" })),
    }),
  },
  {
    name: "swat_squad_update",
    label: "SWAT Squad Update",
    description: "Update an installed squad to the latest marketplace version",
    parameters: Type.Object({
      squad: Type.String({ description: "Squad name to update" }),
    }),
  },
];

async function ensureConnected(logger: any): Promise<Client> {
  if (client) return client;

  transport = new StdioClientTransport({
    command: SWAT_BINARY,
  });

  client = new Client({
    name: "openclaw-swat-bridge",
    version: "2.0.0",
  });

  await client.connect(transport);
  logger.info("SWAT MCP server connected");

  return client;
}

const plugin = {
  id: "swat-mcp-bridge",
  name: "SWAT MCP Bridge",
  description: "Bridge between OpenClaw and SWAT Commander MCP server",

  register(api: OpenClawPluginApi) {
    const logger = api.logger;

    for (const tool of TOOLS) {
      api.registerTool(
        (_ctx) => ({
          name: tool.name,
          label: tool.label,
          description: tool.description,
          parameters: tool.parameters,
          async execute(_toolCallId, params) {
            try {
              const c = await ensureConnected(logger);
              const result = await c.callTool({
                name: tool.name,
                arguments: params as Record<string, unknown>,
              });
              // MCP SDK returns { content: [...] }
              const text = result.content
                ?.filter((c: any) => c.type === "text")
                .map((c: any) => c.text)
                .join("\n") ?? "no result";
              return json({ result: text });
            } catch (err) {
              // If connection died, reset for next call
              client = null;
              transport = null;
              return json({ error: err instanceof Error ? err.message : String(err) });
            }
          },
        }),
        { name: tool.name },
      );
    }

    logger.info("SWAT MCP Bridge registered %d tools", TOOLS.length);
  },
};

export default plugin;
