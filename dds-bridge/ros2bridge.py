import rclpy
from rclpy.node import Node
from std_msgs.msg import String as StringMsg
import json
import requests
import argparse

class ROS2ImageToHTTPBridge(Node):

    def __init__(self, topic, endpoint):
        super().__init__('ros2_image_to_http_bridge')
        self.subscription = self.create_subscription(
            StringMsg,
            topic,
            self.listener_callback,
            10)
        self.endpoint = endpoint

    def listener_callback(self, msg):
        json_data = {
            'image': msg.data
        }
        try:
            response = requests.post(self.endpoint, json=json_data)
            self.get_logger().info(f'Successfully sent data to {self.endpoint}: {response.status_code}')
        except Exception as e:
            self.get_logger().error(f'Failed to send data: {str(e)}')

def main(args=None):
    parser = argparse.ArgumentParser(description='ROS 2 Image to HTTP Bridge')
    parser.add_argument('--topic', type=str, required=True, help='The ROS 2 topic to subscribe to')
    parser.add_argument('--endpoint', type=str, required=True, help='The HTTP endpoint to send JSON data to')
    args = parser.parse_args()

    rclpy.init(args=None)

    ros2_image_to_http_bridge = ROS2ImageToHTTPBridge(args.topic, args.endpoint)

    try:
        rclpy.spin(ros2_image_to_http_bridge)
    except KeyboardInterrupt:
        pass
    finally:
        ros2_image_to_http_bridge.destroy_node()
        rclpy.shutdown()

if __name__ == '__main__':
    main()