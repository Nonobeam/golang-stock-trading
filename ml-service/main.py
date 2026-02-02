"""
ML Prediction Service - Main entry point.
"""

import sys
import os
import argparse

# Add current directory to path
sys.path.insert(0, os.path.dirname(os.path.abspath(__file__)))

from server.grpc_server import serve


def main():
    """Main entry point."""
    parser = argparse.ArgumentParser(description='ML Prediction Service')
    parser.add_argument('--port', type=int, default=50051,
                       help='Port to listen on (default: 50051)')
    
    args = parser.parse_args()
    
    print("=" * 50)
    print("ML Prediction Service")
    print("=" * 50)
    print(f"Listening on port: {args.port}")
    print("Press Ctrl+C to stop")
    print("=" * 50)
    
    serve(port=args.port)


if __name__ == '__main__':
    main()
