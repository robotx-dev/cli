"""
RobotX CLI Python Client

A Python wrapper for the RobotX CLI tool, making it easy to integrate
RobotX deployment capabilities into AI agents and automation scripts.

Usage:
    from robotx_client import RobotXClient

    client = RobotXClient(
        base_url='https://api.robotx.xin',
        api_key='your-api-key'
    )

    result = client.deploy('./my-app', name='my-app', publish=True)
    print(f"Deployed to: {result['url']}")
"""

import subprocess
import json
import os
from typing import Dict, Any, Optional, List
from pathlib import Path


class RobotXError(Exception):
    """Base exception for RobotX errors"""
    pass


class RobotXDeploymentError(RobotXError):
    """Raised when deployment fails"""
    pass


class RobotXAPIError(RobotXError):
    """Raised when API call fails"""
    pass


class RobotXClient:
    """
    Python client for RobotX CLI

    This client wraps the robotx command-line tool and provides a
    Pythonic interface for deploying applications to RobotX.

    Args:
        base_url: RobotX server base URL
        api_key: API key for authentication
        robotx_path: Path to robotx binary (default: 'robotx')
    """

    def __init__(
        self,
        base_url: Optional[str] = None,
        api_key: Optional[str] = None,
        robotx_path: str = 'robotx'
    ):
        self.base_url = base_url or os.getenv('ROBOTX_BASE_URL')
        self.api_key = api_key or os.getenv('ROBOTX_API_KEY')
        self.robotx_path = robotx_path

        # Verify robotx is available
        try:
            self._run_command(['--version'])
        except FileNotFoundError:
            raise RobotXError(
                f"robotx command not found at: {robotx_path}\n"
                "Please install robotx CLI first."
            )

    def _run_command(self, args: List[str]) -> Dict[str, Any]:
        """
        Run a robotx command and return parsed JSON output

        Args:
            args: Command arguments

        Returns:
            Parsed JSON response

        Raises:
            RobotXError: If command fails
        """
        cmd = [self.robotx_path] + args

        # Add global flags if provided
        if self.base_url:
            cmd.extend(['--base-url', self.base_url])
        if self.api_key:
            cmd.extend(['--api-key', self.api_key])

        try:
            result = subprocess.run(
                cmd,
                capture_output=True,
                text=True,
                check=False
            )

            if result.returncode == 0:
                # Success - parse stdout
                if result.stdout.strip():
                    return json.loads(result.stdout)
                return {'success': True}
            else:
                # Error - parse stderr
                try:
                    error_data = json.loads(result.stderr)
                    error_msg = error_data.get('error', 'Unknown error')
                    details = error_data.get('details', '')
                except json.JSONDecodeError:
                    error_msg = result.stderr or 'Command failed'
                    details = ''

                if result.returncode == 3:
                    raise RobotXDeploymentError(f"{error_msg}\n{details}")
                elif result.returncode == 2:
                    raise RobotXAPIError(f"{error_msg}\n{details}")
                else:
                    raise RobotXError(f"{error_msg}\n{details}")

        except FileNotFoundError:
            raise RobotXError(f"robotx command not found: {self.robotx_path}")
        except json.JSONDecodeError as e:
            raise RobotXError(f"Failed to parse command output: {e}")

    def deploy(
        self,
        project_path: str,
        name: Optional[str] = None,
        publish: bool = False,
        wait: bool = True,
        timeout: int = 600,
        visibility: Optional[str] = None
    ) -> Dict[str, Any]:
        """
        Deploy a project to RobotX

        Args:
            project_path: Path to the project directory
            name: Project name (required, deploy by name create-or-update)
            publish: Publish to production after build
            wait: Wait for build to complete
            timeout: Build timeout in seconds
            visibility: Project visibility (public/private)

        Returns:
            Deployment result with project_id, build_id, url, etc.

        Raises:
            RobotXDeploymentError: If deployment fails
            RobotXError: If command fails

        Example:
            >>> client = RobotXClient()
            >>> result = client.deploy('./my-app', name='my-app', publish=True)
            >>> print(result['url'])
            https://my-app.api.robotx.xin
        """
        # Validate inputs
        if not name:
            raise ValueError("'name' must be provided for deploy")

        project_path = str(Path(project_path).resolve())
        if not os.path.exists(project_path):
            raise ValueError(f"Project path does not exist: {project_path}")

        args = ['deploy', project_path]

        args.extend(['--name', name])
        if publish:
            args.append('--publish')
        if not wait:
            args.append('--wait=false')
        if timeout != 600:
            args.extend(['--timeout', str(timeout)])
        if visibility:
            args.extend(['--visibility', visibility])

        return self._run_command(args)

    def status(
        self,
        project_id: Optional[str] = None,
        build_id: Optional[str] = None
    ) -> Dict[str, Any]:
        """
        Get project or build status

        Args:
            project_id: Project ID to check
            build_id: Build ID to check

        Returns:
            Status information

        Raises:
            RobotXAPIError: If API call fails

        Example:
            >>> status = client.status(project_id='proj_123')
            >>> print(status['status'])
            running
        """
        if not project_id and not build_id:
            raise ValueError("Either 'project_id' or 'build_id' must be provided")

        args = ['status']

        if project_id:
            args.extend(['--project-id', project_id])
        if build_id:
            args.extend(['--build-id', build_id])

        return self._run_command(args)

    def logs(self, build_id: str) -> str:
        """
        Deprecated. RobotX no longer provides remote build logs.
        """
        _ = build_id
        raise RobotXError('RobotX no longer provides remote build logs')

    def publish(self, project_id: str, build_id: str) -> Dict[str, Any]:
        """
        Publish a build to production

        Args:
            project_id: Project ID
            build_id: Build ID to publish

        Returns:
            Publish result

        Raises:
            RobotXAPIError: If API call fails

        Example:
            >>> result = client.publish('proj_123', 'build_123')
            >>> print(result['url'])
            https://my-app.api.robotx.xin
        """
        return self._run_command([
            'publish',
            '--project-id', project_id,
            '--build-id', build_id,
        ])

    def versions(self, project_id: str, limit: int = 20) -> Dict[str, Any]:
        """
        List recent build versions for a project
        """
        if not project_id:
            raise ValueError("'project_id' must be provided")
        args = ['versions', '--project-id', project_id]
        if limit != 20:
            args.extend(['--limit', str(limit)])
        return self._run_command(args)

    def projects(self, limit: int = 50) -> Dict[str, Any]:
        """
        List projects for the current account
        """
        args = ['projects']
        if limit != 50:
            args.extend(['--limit', str(limit)])
        return self._run_command(args)

    def wait_for_build(
        self,
        build_id: str,
        timeout: int = 600,
        poll_interval: int = 5
    ) -> Dict[str, Any]:
        """
        Wait for a build to complete

        Args:
            build_id: Build ID to wait for
            timeout: Maximum time to wait in seconds
            poll_interval: Time between status checks in seconds

        Returns:
            Final build status

        Raises:
            RobotXDeploymentError: If build fails
            TimeoutError: If build doesn't complete within timeout

        Example:
            >>> result = client.deploy('./app', name='app', wait=False)
            >>> final_status = client.wait_for_build(result['build_id'])
        """
        import time

        start_time = time.time()

        while True:
            status = self.status(build_id=build_id)
            build_status = status.get('build', {}).get('status')

            if build_status in ['success', 'completed']:
                return status
            elif build_status in ['failed', 'error']:
                raise RobotXDeploymentError(f"Build failed: {build_status}")

            elapsed = time.time() - start_time
            if elapsed > timeout:
                raise TimeoutError(f"Build did not complete within {timeout}s")

            time.sleep(poll_interval)


# Convenience functions for quick usage

def deploy(
    project_path: str,
    name: str,
    base_url: Optional[str] = None,
    api_key: Optional[str] = None,
    **kwargs
) -> Dict[str, Any]:
    """
    Quick deploy function

    Example:
        >>> from robotx_client import deploy
        >>> result = deploy('./my-app', 'my-app', publish=True)
    """
    client = RobotXClient(base_url=base_url, api_key=api_key)
    return client.deploy(project_path, name=name, **kwargs)


def status(
    project_id: Optional[str] = None,
    build_id: Optional[str] = None,
    base_url: Optional[str] = None,
    api_key: Optional[str] = None
) -> Dict[str, Any]:
    """
    Quick status check function

    Example:
        >>> from robotx_client import status
        >>> result = status(project_id='proj_123')
    """
    client = RobotXClient(base_url=base_url, api_key=api_key)
    return client.status(project_id=project_id, build_id=build_id)


if __name__ == '__main__':
    # Example usage
    import sys

    if len(sys.argv) < 3:
        print("Usage: python robotx_client.py <project_path> <project_name>")
        sys.exit(1)

    project_path = sys.argv[1]
    project_name = sys.argv[2]

    try:
        client = RobotXClient()
        print(f"Deploying {project_path} as {project_name}...")

        result = client.deploy(
            project_path=project_path,
            name=project_name,
            publish=True
        )

        print(f"✅ Deployment successful!")
        print(f"📦 Project ID: {result['project_id']}")
        print(f"🔨 Build ID: {result['build_id']}")
        print(f"🌐 URL: {result['url']}")

    except RobotXError as e:
        print(f"❌ Deployment failed: {e}", file=sys.stderr)
        sys.exit(1)
