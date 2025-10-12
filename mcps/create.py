#!/usr/bin/env python3

import os
import json
import shutil
import argparse
import logging
from pathlib import Path
from typing import Dict, Any

logger = logging.getLogger(__name__)

class MCPServerGenerator:
    """Generator for creating new MCP servers from templates."""

    def __init__(self, template_dir: Path = None, output_dir: Path = None):
        """Initialize the MCP server generator.

        Args:
            template_dir: Directory containing server templates
            output_dir: Directory where new servers will be created
        """
        if template_dir is None:
            template_dir = Path(__file__).parent / "template"
        if output_dir is None:
            output_dir = Path(__file__).parent

        self.template_dir = template_dir
        self.output_dir = output_dir

    def create_server(self, name: str, description: str = None, author: str = None) -> Path:
        """Create a new MCP server from template.

        Args:
            name: Name of the new server
            description: Description of the server
            author: Author name

        Returns:
            Path to the created server directory
        """
        if not self.template_dir.exists():
            raise FileNotFoundError(f"Template directory not found: {self.template_dir}")

        # Sanitize server name
        sanitized_name = self._sanitize_name(name)
        server_dir = self.output_dir / sanitized_name

        if server_dir.exists():
            raise FileExistsError(f"Server directory already exists: {server_dir}")

        logger.info(f"Creating MCP server '{name}' at {server_dir}")

        # Create server directory
        server_dir.mkdir(parents=True)

        # Copy template files
        self._copy_template_files(server_dir)

        # Customize manifest
        self._customize_manifest(server_dir, name, description, author)

        # Customize server.py
        self._customize_server_file(server_dir, name, sanitized_name)

        logger.info(f"Successfully created MCP server: {server_dir}")
        return server_dir

    def _sanitize_name(self, name: str) -> str:
        """Sanitize server name for use as directory and Python module name.

        Args:
            name: Original server name

        Returns:
            Sanitized name
        """
        # Replace spaces and special characters with underscores
        import re
        sanitized = re.sub(r'[^a-zA-Z0-9_]', '_', name.lower())

        # Remove consecutive underscores
        sanitized = re.sub(r'_+', '_', sanitized)

        # Remove leading/trailing underscores
        sanitized = sanitized.strip('_')

        # Ensure it doesn't start with a number
        if sanitized and sanitized[0].isdigit():
            sanitized = f"mcp_{sanitized}"

        return sanitized or "mcp_server"

    def _copy_template_files(self, server_dir: Path):
        """Copy template files to new server directory.

        Args:
            server_dir: Target server directory
        """
        for item in self.template_dir.iterdir():
            if item.is_file():
                shutil.copy2(item, server_dir)
            elif item.is_dir():
                shutil.copytree(item, server_dir / item.name)

    def _customize_manifest(self, server_dir: Path, name: str, description: str = None, author: str = None):
        """Customize the mcp-server.json manifest file.

        Args:
            server_dir: Server directory
            name: Server name
            description: Server description
            author: Author name
        """
        manifest_path = server_dir / "mcp-server.json"

        with open(manifest_path, 'r', encoding='utf-8') as f:
            manifest = json.load(f)

        # Update manifest fields
        manifest["name"] = name
        if description:
            manifest["description"] = description
        if author:
            manifest["author"] = author

        with open(manifest_path, 'w', encoding='utf-8') as f:
            json.dump(manifest, f, indent=2, ensure_ascii=False)

    def _customize_server_file(self, server_dir: Path, name: str, sanitized_name: str):
        """Customize the server.py file.

        Args:
            server_dir: Server directory
            name: Original server name
            sanitized_name: Sanitized server name
        """
        server_file = server_dir / "server.py"

        with open(server_file, 'r', encoding='utf-8') as f:
            content = f.read()

        # Replace template placeholders
        content = content.replace('template-mcp-server', name)
        content = content.replace('app = Server("template-mcp-server")', f'app = Server("{name}")')

        with open(server_file, 'w', encoding='utf-8') as f:
            f.write(content)

    def list_templates(self) -> Dict[str, Any]:
        """List available server templates.

        Returns:
            Dictionary with template information
        """
        templates = {}

        if not self.template_dir.exists():
            return templates

        for item in self.template_dir.iterdir():
            if item.is_dir():
                manifest_path = item / "mcp-server.json"
                if manifest_path.exists():
                    try:
                        with open(manifest_path, 'r', encoding='utf-8') as f:
                            manifest = json.load(f)
                        templates[item.name] = {
                            "name": manifest.get("name", item.name),
                            "description": manifest.get("description", ""),
                            "version": manifest.get("version", "unknown")
                        }
                    except Exception as e:
                        logger.warning(f"Error reading template {item.name}: {e}")

        return templates

def main():
    """Main CLI entry point."""
    parser = argparse.ArgumentParser(
        description="Create new MCP servers from templates",
        prog="python -m mcps.create"
    )

    parser.add_argument(
        "name",
        help="Name of the new MCP server"
    )

    parser.add_argument(
        "--description", "-d",
        help="Description of the MCP server"
    )

    parser.add_argument(
        "--author", "-a",
        help="Author name"
    )

    parser.add_argument(
        "--output-dir", "-o",
        type=Path,
        help="Output directory (default: current mcps/ directory)"
    )

    parser.add_argument(
        "--template-dir", "-t",
        type=Path,
        help="Template directory (default: mcps/template/)"
    )

    parser.add_argument(
        "--list-templates", "-l",
        action="store_true",
        help="List available templates"
    )

    parser.add_argument(
        "--verbose", "-v",
        action="store_true",
        help="Enable verbose logging"
    )

    args = parser.parse_args()

    # Setup logging
    level = logging.DEBUG if args.verbose else logging.INFO
    logging.basicConfig(
        level=level,
        format='%(asctime)s - %(levelname)s - %(message)s'
    )

    try:
        generator = MCPServerGenerator(
            template_dir=args.template_dir,
            output_dir=args.output_dir
        )

        if args.list_templates:
            templates = generator.list_templates()
            if templates:
                print("Available templates:")
                for template_name, info in templates.items():
                    print(f"  {template_name}: {info['description']}")
            else:
                print("No templates found.")
            return

        # Create new server
        server_dir = generator.create_server(
            name=args.name,
            description=args.description,
            author=args.author
        )

        print(f"Successfully created MCP server: {server_dir}")
        print("\nNext steps:")
        print(f"1. cd {server_dir}")
        print("2. Install dependencies: pip install -r requirements.txt")
        print("3. Edit server.py to implement your tools")
        print("4. Test the server: python server.py")

    except Exception as e:
        logger.error(f"Error creating MCP server: {e}")
        return 1

    return 0

if __name__ == "__main__":
    exit(main())