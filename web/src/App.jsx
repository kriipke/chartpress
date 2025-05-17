import "./styles.css";
import './SubchartConfigurator.css';

import SubchartConfigurator from './components/SubchartConfigurator';

function App() {
  return (
    <div>
      {/* other content */}
      <SubchartConfigurator />
    </div>
  );
}
export default App;

import { ChartpressHelmChartWizard } from "./ChartpressHelmChartWizard/ChartpressHelmChartWizard";

export default function App() {
  return (
    <div>
      <ChartpressHelmChartWizard />
    </div>
  );
}

