import pandas as pd
import json, argparse, tabulate, sys
import matplotlib.pyplot as plt

# Parse command-line arguments
parser = argparse.ArgumentParser(description='Parse and analyze HAR files')
parser.add_argument("-f", "--file", help="path to.har file")
parser.add_argument("-s", "--status", help="status code range to include (e.g. 200-299)")
parser.add_argument("-c", "--content", help="content type to include (e.g. text/html)")
parser.add_argument("-g", "--group", help="column to group by (e.g. url)")
parser.add_argument("-o", "--output", help="output file path if not present will use STDOUT as output")
parser.add_argument("--summarize", action="store_true", help="summarizes the HAR file and returns json")
parser.add_argument("--plot", action="store_true", help="generate analyses graph")
args = parser.parse_args()

# Check if no arguments were passed
if len(sys.argv) < 2:
    parser.print_help()
    print("\nExample usage: python har_parser.py -f example.har --summarize")
    sys.exit(1)

# Load the.har file as a dictionary
with open(args.file, 'r') as f:
    har_data = json.load(f)

# Extract the entries from the HAR data
entries = har_data['log']['entries']

# Create a list of dictionaries containing information about each request
requests = []
for entry in entries:
    request = {
        'url': entry['request']['url'],
        'method': entry['request']['method'],
        'status': entry['response']['status'],
        'time': entry['time'],
        'size': entry.get('response', {}).get('content', {}).get('size', 0),
        'mime_type': entry.get('response', {}).get('content', {}).get('mimeType', ''),
        'error': entry['response'].get('_error', '')
    }
    requests.append(request)

# Create Failed Count
failed = 0
for entry in entries:
    if entry['response']['status'] != 200:
        failed += 1

# Convert the list of dictionaries to a pandas DataFrame
df = pd.DataFrame(requests)

# Filter the DataFrame by status code and content type
if args.status:
    status_range = [int(x) for x in args.status.split('-')]
    df = df[(df['status'] >= status_range[0]) & (df['status'] <= status_range[1])]
if args.content:
    df = df[df['mime_type'].str.contains(args.content)]

# Group the DataFrame by the specified column
if args.group:
    df = df.groupby(args.group).size().reset_index(name='count')

# Sort the DataFrame by the time column
df = df.sort_values(by='time')

# Print the DataFrame as a table
if args.output == None and args.summarize != True:
    table = tabulate.tabulate(df, headers='keys', tablefmt='psql', showindex=False)
    print(table)

# Generate a plot of the request progression over time
if args.plot:
    # Calculate the cumulative sum of request times
    cumulative_sum = df['time'].cumsum()

    # Create a figure and a set of subplots
    fig, axs = plt.subplots(2, 3, figsize=(12, 6))

    # Plot the cumulative sum
    axs[0, 0].plot(cumulative_sum)
    axs[0, 0].set_xlabel('Request')
    axs[0, 0].set_ylabel('Time (ms)')
    axs[0, 0].set_title('Request Progression')

    # Plot the status codes
    status_counts = df['status'].value_counts()
    axs[0, 1].bar(status_counts.index, status_counts.values)
    axs[0, 1].set_xlabel('Status Code')
    axs[0, 1].set_ylabel('Count')
    axs[0, 1].set_title('Status Codes')

    # Plot the request sizes
    axs[0, 2].plot(df['size'])
    axs[0, 2].set_xlabel('Request')
    axs[0, 2].set_ylabel('Size (bytes)')
    axs[0, 2].set_title('Request Sizes')

    # Plot the request time vs. size
    axs[1, 0].scatter(df['time'], df['size'])
    axs[1, 0].set_xlabel('Time (ms)')
    axs[1, 0].set_ylabel('Size (bytes)')
    axs[1, 0].set_title('Request Time vs. Size')

    # Plot the request times by status code
    axs[1, 1].boxplot(df.groupby('status')['time'].apply(list))
    axs[1, 1].set_xlabel('Status Code')
    axs[1, 1].set_ylabel('Time (ms)')
    axs[1, 1].set_title('Request Times by Status Code')

    # Plot the request times histogram
    axs[1, 2].hist(df['time'], bins=50)
    axs[1, 2].set_xlabel('Time (ms)')
    axs[1, 2].set_ylabel('Count')
    axs[1, 2].set_title('Request Times Histogram')

    # Adjust the spacing between subplots
    plt.subplots_adjust(wspace=0.3, hspace=0.4)

    # Show the plot
    plt.show()

# Generate a summary of the data
summary = {
    'total_requests': len(df),
    'failed_request': failed,
    'average_time': df['time'].mean(),
    'fastest_request': df['time'].min(),
    'slowest_request': df['time'].max(),
    'status_counts': df['status'].value_counts().to_dict(),
    'content_types': df['mime_type'].value_counts().to_dict()
}

# Summarize the HAR file and output nice json!
if args.summarize:
    print(json.dumps(summary, indent=4))

# Export the DataFrame to a file
if args.output:
    print(json.dumps(summary, indent=4))
    df.to_csv(args.output, index=False)