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
    description: "Dispatch a new task to a SWAT squad",
    parameters: Type.Object({
      brief: Type.String({ description: "Task description" }),
      details: Type.Optional(Type.String({ description: "Additional details" })),
      squad: Type.Optional(Type.String({ description: "Target squad (auto-classify if omitted)" })),
    }),
  },
  {
    name: "swat_status",
    label: "SWAT Status",
    description: "Get SWAT task status and unnotified completions",
    parameters: Type.Object({}),
  },
  {
    name: "swat_list",
    label: "SWAT List",
    description: "List all SWAT operations",
    parameters: Type.Object({
      status: Type.Optional(Type.String({ description: "Filter by status (queued/active/completed/failed)" })),
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
    description: "Create a scheduled task",
    parameters: Type.Object({
      brief: Type.String({ description: "Task description" }),
      cron: Type.String({ description: "Cron expression" }),
      details: Type.Optional(Type.String({ description: "Additional details" })),
      squad: Type.Optional(Type.String({ description: "Target squad" })),
    }),
  },
  {
    name: "swat_browse",
    label: "SWAT Browse",
    description: "List all squads available in the marketplace",
    parameters: Type.Object({}),
  },
  {
    name: "swat_install",
    label: "SWAT Install",
    description: "Install a squad from the marketplace",
    parameters: Type.Object({
      squad: Type.String({ description: "Squad name to install" }),
    }),
  },
  {
    name: "swat_uninstall",
    label: "SWAT Uninstall",
    description: "Uninstall a squad and clean up orphaned dependencies",
    parameters: Type.Object({
      squad: Type.String({ description: "Squad name to uninstall" }),
      purge: Type.Optional(Type.Boolean({ description: "Also delete runtime data (default: false)" })),
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
              return json((() => { try { return JSON.parse(text); } catch { return { result: text }; } })());
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
