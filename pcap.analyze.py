from scapy.all import *

# Replace 'filename.pcap' with the name of your pcap file
packets = rdpcap('tls.pcapng')

# Initialize counters for different types of packets
total_packets = 0
tcp_packets = 0
udp_packets = 0
icmp_packets = 0
failed_transmissions = 0

# Initialize dictionaries to keep track of connections and data
tcp_connections = {}
udp_connections = {}
other_data = {}

# Loop through each packet in the pcap file
for packet in packets:
    total_packets += 1
    if packet.haslayer(TCP):
        tcp_packets += 1
        src_ip = packet[IP].src
        dst_ip = packet[IP].dst
        src_port = packet[TCP].sport
        dst_port = packet[TCP].dport
        flags = packet[TCP].flags
        if flags & 0x02:
            # SYN packet, start of a new connection
            tcp_connections[(src_ip, src_port, dst_ip, dst_port)] = {'syn': packet}
        elif flags & 0x10:
            # ACK packet, end of a connection
            if (dst_ip, dst_port, src_ip, src_port) in tcp_connections:
                tcp_connections[(dst_ip, dst_port, src_ip, src_port)]['ack'] = packet
            else:
                tcp_connections[(src_ip, src_port, dst_ip, dst_port)] = {'ack': packet}
        elif flags & 0x04:
            # RST packet, failed transmission
            failed_transmissions += 1
            if (dst_ip, dst_port, src_ip, src_port) in tcp_connections:
                tcp_connections[(dst_ip, dst_port, src_ip, src_port)]['rst'] = packet
            else:
                tcp_connections[(src_ip, src_port, dst_ip, dst_port)] = {'rst': packet}
        elif flags & 0x01:
            # FIN packet, end of a connection
            if (dst_ip, dst_port, src_ip, src_port) in tcp_connections:
                tcp_connections[(dst_ip, dst_port, src_ip, src_port)]['fin'] = packet
            else:
                tcp_connections[(src_ip, src_port, dst_ip, dst_port)] = {'fin': packet}
        elif flags & 0x18:
            # TLS handshake
            if (dst_ip, dst_port, src_ip, src_port) in tcp_connections:
                if 'tls' not in tcp_connections[(dst_ip, dst_port, src_ip, src_port)]:
                    tcp_connections[(dst_ip, dst_port, src_ip, src_port)]['tls'] = []
                tcp_connections[(dst_ip, dst_port, src_ip, src_port)]['tls'].append(packet)
            else:
                if 'tls' not in tcp_connections[(src_ip, src_port, dst_ip, dst_port)]:
                    tcp_connections[(src_ip, src_port, dst_ip, dst_port)]['tls'] = []
                tcp_connections[(src_ip, src_port, dst_ip, dst_port)]['tls'].append(packet)
    elif packet.haslayer(UDP):
        udp_packets += 1
        src_ip = packet[IP].src
        dst_ip = packet[IP].dst
        src_port = packet[UDP].sport
        dst_port = packet[UDP].dport
        if (src_ip, src_port, dst_ip, dst_port) not in udp_connections:
            udp_connections[(src_ip, src_port, dst_ip, dst_port)] = []
        udp_connections[(src_ip, src_port, dst_ip, dst_port)].append(packet)
    else:
        # Other data, store in dictionary
        if packet.haslayer(Raw):
            data = packet[Raw].load
        else:
            data = ''
        if packet.haslayer(IP):
            src_ip = packet[IP].src
            dst_ip = packet[IP].dst
        else:
            src_ip = ''
            dst_ip = ''
        if (src_ip, dst_ip) not in other_data:
            other_data[(src_ip, dst_ip)] = []
        other_data[(src_ip, dst_ip)].append(data)

# Print out a summary of the network traffic
print(f'Total packets: {total_packets}')
print(f'TCP packets: {tcp_packets}')
print(f'UDP packets: {udp_packets}')
print(f'ICMP packets: {icmp_packets}')
print(f'Failed transmissions: {failed_transmissions}')

# Print out information about TCP connections
print('TCP connections:')
for connection, packets in tcp_connections.items():
    if 'syn' in packets and 'ack' in packets:
        print(f'{connection[0]}:{connection[1]} -> {connection[2]}:{connection[3]}')
        if 'tls' in packets:
            print('  TLS handshake:')
            for packet in packets['tls']:
                print(f'    {packet.summary()}')
    elif 'syn' in packets:
        print(f'{connection[0]}:{connection[1]} -> {connection[2]}:{connection[3]} (SYN)')
    elif 'ack' in packets:
        print(f'{connection[0]}:{connection[1]} -> {connection[2]}:{connection[3]} (ACK)')
    elif 'rst' in packets:
        print(f'{connection[0]}:{connection[1]} -> {connection[2]}:{connection[3]} (RST)')
    elif 'fin' in packets:
        print(f'{connection[0]}:{connection[1]} -> {connection[2]}:{connection[3]} (FIN)')

# Print out information about UDP connections
print('UDP connections:')
for connection, packets in udp_connections.items():
    print(f'{connection[0]}:{connection[1]} -> {connection[2]}:{connection[3]} ({len(packets)} packets)')

# Print out other data
print('Other data:')
for connection, data in other_data.items():
    print(f'{connection[0]} -> {connection[1]} ({len(data)} packets)')
    for d in data:
        print(d)

# Print out a tree view of all connections
print('Connection tree:')
for connection, packets in tcp_connections.items():
    if 'syn' in packets and 'ack' in packets:
        print(f'{connection[0]}:{connection[1]} -> {connection[2]}:{connection[3]}')
        for sub_connection, sub_packets in tcp_connections.items():
            if 'syn' in sub_packets and 'ack' in sub_packets and sub_connection[0] == connection[2] and sub_connection[2] == connection[0]:
                print(f'  {sub_connection[0]}:{sub_connection[1]} -> {sub_connection[2]}:{sub_connection[3]}')