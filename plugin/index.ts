import type { OpenClawPluginApi } from "openclaw/plugin-sdk";
import { Client } from "@modelcontextprotocol/sdk/client/index.js";
import { StdioClientTransport } from "@modelcontextprotocol/sdk/client/stdio.js";
import { Type } from "@sinclair/typebox";

let client: Client | null = null;
let transport: StdioClientTransport | null = null;

function json(data: unknown) {
  return {
    content: [{ type: "text" as const, text: JSON.stringify(data, null, 2) }],
    details: data,
  };
}

let swatBinary = "";

const TOOLS = [
  {
    name: "swat_dispatch",
    label: "SWAT Dispatch",
    description: "Dispatch a new task to a SWAT squad. Squad is auto-classified. Returns immediately; task runs in background. Read the swat skill for dispatch workflow and completion monitoring guidance.",
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
    name: "swat_schedule_create",
    label: "SWAT Schedule Create",
    description: "Create a scheduled recurring task. Zero LLM cost. Read the swat skill for scheduling guidance.",
    parameters: Type.Object({
      brief: Type.String({ description: "Task description" }),
      cron: Type.String({ description: "Cron expression, 5-field: min hour dom month dow" }),
      details: Type.Optional(Type.String({ description: "Additional details" })),
      timezone: Type.Optional(Type.String({ description: "IANA timezone, e.g. Asia/Shanghai (default: UTC)" })),
      immediate: Type.Optional(Type.Boolean({ description: "If true, trigger first run immediately (default: false)" })),
    }),
  },
  {
    name: "swat_schedules",
    label: "SWAT Schedules",
    description: "List all scheduled tasks with next run times",
    parameters: Type.Object({}),
  },
  {
    name: "swat_schedule_delete",
    label: "SWAT Schedule Delete",
    description: "Delete a scheduled task",
    parameters: Type.Object({
      id: Type.String({ description: "Schedule ID" }),
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
    description: "Install a squad from the marketplace. Returns prerequisite warnings if any dependent skills need setup.",
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
    command: swatBinary,
  });

  client = new Client({
    name: "openclaw-swat-bridge",
    version: "1.0.0",
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
    if (api.pluginConfig?.binaryPath) {
      swatBinary = api.pluginConfig.binaryPath as string;
    }
    if (!swatBinary) {
      logger.error("SWAT binary path not configured. Run install.sh or set plugins.entries.swat-mcp-bridge.config.binaryPath");
      return;
    }

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

    logger.info(`SWAT MCP Bridge registered ${TOOLS.length} tools`);
  },
};

export default plugin;
