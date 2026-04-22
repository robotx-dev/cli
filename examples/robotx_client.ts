/**
 * RobotX CLI Node.js/TypeScript Client
 *
 * A TypeScript/JavaScript wrapper for the RobotX CLI tool, making it easy
 * to integrate RobotX deployment capabilities into AI agents and automation scripts.
 *
 * @example
 * ```typescript
 * import { RobotXClient } from './robotx_client';
 *
 * const client = new RobotXClient({
 *   baseUrl: 'https://api.robotx.xin',
 *   apiKey: 'your-api-key'
 * });
 *
 * const result = await client.deploy('./my-app', { name: 'my-app', publish: true });
 * console.log(`Deployed to: ${result.url}`);
 * ```
 */

import { exec } from 'child_process';
import { promisify } from 'util';
import { existsSync } from 'fs';
import { resolve } from 'path';

const execAsync = promisify(exec);

/**
 * RobotX client configuration
 */
export interface RobotXConfig {
  /** RobotX server base URL */
  baseUrl?: string;
  /** API key for authentication */
  apiKey?: string;
  /** Path to robotx binary (default: 'robotx') */
  robotxPath?: string;
}

/**
 * Deploy options
 */
export interface DeployOptions {
  /** Project name (required, deploy by name create-or-update) */
  name?: string;
  /** Publish to production after build */
  publish?: boolean;
  /** Wait for build to complete */
  wait?: boolean;
  /** Build timeout in seconds */
  timeout?: number;
  /** Project visibility (public/private) */
  visibility?: 'public' | 'private';
}

/**
 * Deploy result
 */
export interface DeployResult {
  success: boolean;
  project_id: string;
  build_id: string;
  status: string;
  url: string;
  message: string;
}

/**
 * Status result
 */
export interface StatusResult {
  success: boolean;
  status: string;
  project?: {
    id: string;
    name: string;
    visibility: string;
  };
  build?: {
    id: string;
    status: string;
    created_at: string;
  };
}

/**
 * RobotX error class
 */
export class RobotXError extends Error {
  constructor(message: string, public details?: string) {
    super(message);
    this.name = 'RobotXError';
  }
}

/**
 * RobotX deployment error
 */
export class RobotXDeploymentError extends RobotXError {
  constructor(message: string, details?: string) {
    super(message, details);
    this.name = 'RobotXDeploymentError';
  }
}

/**
 * RobotX API error
 */
export class RobotXAPIError extends RobotXError {
  constructor(message: string, details?: string) {
    super(message, details);
    this.name = 'RobotXAPIError';
  }
}

/**
 * RobotX CLI Client
 *
 * This client wraps the robotx command-line tool and provides a
 * TypeScript/JavaScript interface for deploying applications to RobotX.
 */
export class RobotXClient {
  private baseUrl?: string;
  private apiKey?: string;
  private robotxPath: string;

  constructor(config: RobotXConfig = {}) {
    this.baseUrl = config.baseUrl || process.env.ROBOTX_BASE_URL;
    this.apiKey = config.apiKey || process.env.ROBOTX_API_KEY;
    this.robotxPath = config.robotxPath || 'robotx';
  }

  /**
   * Run a robotx command and return parsed JSON output
   */
  private async runCommand(args: string[]): Promise<any> {
    const cmd = [this.robotxPath, ...args];

    // Add global flags if provided
    if (this.baseUrl) {
      cmd.push('--base-url', this.baseUrl);
    }
    if (this.apiKey) {
      cmd.push('--api-key', this.apiKey);
    }

    try {
      const { stdout } = await execAsync(cmd.join(' '));

      if (stdout.trim()) {
        return JSON.parse(stdout);
      }
      return { success: true };
    } catch (error: any) {
      // Parse error from stderr
      let errorData: any;
      try {
        errorData = JSON.parse(error.stderr || '{}');
      } catch {
        throw new RobotXError(error.message);
      }

      const errorMsg = errorData.error || 'Unknown error';
      const details = errorData.details || '';

      if (error.code === 3) {
        throw new RobotXDeploymentError(errorMsg, details);
      } else if (error.code === 2) {
        throw new RobotXAPIError(errorMsg, details);
      } else {
        throw new RobotXError(errorMsg, details);
      }
    }
  }

  /**
   * Deploy a project to RobotX
   *
   * @param projectPath - Path to the project directory
   * @param options - Deploy options
   * @returns Deployment result
   *
   * @example
   * ```typescript
   * const result = await client.deploy('./my-app', {
   *   name: 'my-app',
   *   publish: true
   * });
   * console.log(`Deployed to: ${result.url}`);
   * ```
   */
  async deploy(
    projectPath: string,
    options: DeployOptions = {}
  ): Promise<DeployResult> {
    // Validate inputs
    if (!options.name) {
      throw new Error("'name' must be provided for deploy");
    }

    const resolvedPath = resolve(projectPath);
    if (!existsSync(resolvedPath)) {
      throw new Error(`Project path does not exist: ${resolvedPath}`);
    }

    const args = ['deploy', resolvedPath];

    args.push('--name', options.name);
    if (options.publish) {
      args.push('--publish');
    }
    if (options.wait === false) {
      args.push('--wait=false');
    }
    if (options.timeout) {
      args.push('--timeout', options.timeout.toString());
    }
    if (options.visibility) {
      args.push('--visibility', options.visibility);
    }

    return this.runCommand(args);
  }

  /**
   * Get project or build status
   *
   * @param projectId - Project ID to check
   * @param buildId - Build ID to check
   * @returns Status information
   *
   * @example
   * ```typescript
   * const status = await client.status({ projectId: 'proj_123' });
   * console.log(status.status);
   * ```
   */
  async status(options: {
    projectId?: string;
    buildId?: string;
  }): Promise<StatusResult> {
    if (!options.projectId && !options.buildId) {
      throw new Error("Either 'projectId' or 'buildId' must be provided");
    }

    const args = ['status'];

    if (options.projectId) {
      args.push('--project-id', options.projectId);
    }
    if (options.buildId) {
      args.push('--build-id', options.buildId);
    }

    return this.runCommand(args);
  }

  /**
   * Deprecated. RobotX no longer provides remote build logs.
   */
  async logs(buildId: string): Promise<string> {
    void buildId;
    throw new Error('RobotX no longer provides remote build logs');
  }

  /**
   * Publish a build to production
   *
   * @param projectId - Project ID
   * @param buildId - Build ID to publish
   * @returns Publish result
   *
   * @example
   * ```typescript
   * const result = await client.publish('proj_123', 'build_123');
   * console.log(`Published to: ${result.url}`);
   * ```
   */
  async publish(projectId: string, buildId: string): Promise<any> {
    return this.runCommand([
      'publish',
      '--project-id', projectId,
      '--build-id', buildId,
    ]);
  }

  /**
   * List recent build versions for a project
   *
   * @param projectId - Project ID
   * @param limit - Max number of versions
   * @returns Version list
   *
   * @example
   * ```typescript
   * const result = await client.versions('proj_123', 20);
   * console.log(result.builds);
   * ```
   */
  async versions(projectId: string, limit: number = 20): Promise<any> {
    return this.runCommand([
      'versions',
      '--project-id', projectId,
      '--limit', String(limit),
    ]);
  }

  /**
   * List projects for the current account
   *
   * @param limit - Max number of projects
   * @returns Project list
   */
  async projects(limit: number = 50): Promise<any> {
    return this.runCommand([
      'projects',
      '--limit', String(limit),
    ]);
  }

  /**
   * Wait for a build to complete
   *
   * @param buildId - Build ID to wait for
   * @param timeout - Maximum time to wait in seconds
   * @param pollInterval - Time between status checks in seconds
   * @returns Final build status
   *
   * @example
   * ```typescript
   * const result = await client.deploy('./app', { name: 'app', wait: false });
   * const finalStatus = await client.waitForBuild(result.build_id);
   * ```
   */
  async waitForBuild(
    buildId: string,
    timeout: number = 600,
    pollInterval: number = 5
  ): Promise<StatusResult> {
    const startTime = Date.now();

    while (true) {
      const status = await this.status({ buildId });
      const buildStatus = status.build?.status;

      if (buildStatus === 'success' || buildStatus === 'completed') {
        return status;
      } else if (buildStatus === 'failed' || buildStatus === 'error') {
        throw new RobotXDeploymentError(`Build failed: ${buildStatus}`);
      }

      const elapsed = (Date.now() - startTime) / 1000;
      if (elapsed > timeout) {
        throw new Error(`Build did not complete within ${timeout}s`);
      }

      await new Promise(resolve => setTimeout(resolve, pollInterval * 1000));
    }
  }
}

/**
 * Quick deploy function
 *
 * @example
 * ```typescript
 * import { deploy } from './robotx_client';
 *
 * const result = await deploy('./my-app', 'my-app', { publish: true });
 * console.log(`Deployed to: ${result.url}`);
 * ```
 */
export async function deploy(
  projectPath: string,
  name: string,
  options: Omit<DeployOptions, 'name'> & RobotXConfig = {}
): Promise<DeployResult> {
  const { baseUrl, apiKey, robotxPath, ...deployOptions } = options;
  const client = new RobotXClient({ baseUrl, apiKey, robotxPath });
  return client.deploy(projectPath, { ...deployOptions, name });
}

/**
 * Quick status check function
 *
 * @example
 * ```typescript
 * import { status } from './robotx_client';
 *
 * const result = await status({ projectId: 'proj_123' });
 * console.log(result.status);
 * ```
 */
export async function status(
  options: { projectId?: string; buildId?: string } & RobotXConfig
): Promise<StatusResult> {
  const { baseUrl, apiKey, robotxPath, ...statusOptions } = options;
  const client = new RobotXClient({ baseUrl, apiKey, robotxPath });
  return client.status(statusOptions);
}

// Example usage
if (require.main === module) {
  const [, , projectPath, projectName] = process.argv;

  if (!projectPath || !projectName) {
    console.error('Usage: ts-node robotx_client.ts <project_path> <project_name>');
    process.exit(1);
  }

  (async () => {
    try {
      const client = new RobotXClient();
      console.log(`Deploying ${projectPath} as ${projectName}...`);

      const result = await client.deploy(projectPath, {
        name: projectName,
        publish: true
      });

      console.log('✅ Deployment successful!');
      console.log(`📦 Project ID: ${result.project_id}`);
      console.log(`🔨 Build ID: ${result.build_id}`);
      console.log(`🌐 URL: ${result.url}`);
    } catch (error) {
      if (error instanceof RobotXError) {
        console.error(`❌ Deployment failed: ${error.message}`);
        if (error.details) {
          console.error(`Details: ${error.details}`);
        }
      } else {
        console.error(`❌ Unexpected error:`, error);
      }
      process.exit(1);
    }
  })();
}
