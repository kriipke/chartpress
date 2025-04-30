import YAML from 'yaml';
import chartpressRaw from 'chartpress.yaml?raw';

const chartpressConfig = YAML.parse(chartpressRaw);

export default chartpressConfig;
