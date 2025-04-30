import React, { useState } from 'react';
import Slider from 'react-slick';
import { BrowserRouter as Router, Route, Routes, Link } from 'react-router-dom';
import 'slick-carousel/slick/slick.css';
import 'slick-carousel/slick/slick-theme.css';
import './App.css'; // Add custom styles here
import Documentation from './Documentation';


function App() {
  const [umbrellaChartName, setUmbrellaChartName] = useState('');
  const [subcharts, setSubcharts] = useState([{ name: '', workload: 'deployment' }]);
  const [currentStep, setCurrentStep] = useState(0);
  const [downloadUrl, setDownloadUrl] = useState('');

  const handleAddSubchart = () => {
    setSubcharts([...subcharts, { name: '', workload: 'deployment' }]);
  };

  const handleSubchartChange = (index, field, value) => {
    const updatedSubcharts = [...subcharts];
    updatedSubcharts[index][field] = value;
    setSubcharts(updatedSubcharts);
  };

  const generateTreeStructure = () => {
    let tree = '';
    tree += `${umbrellaChartName || 'umbrella-chart'}/\n`;
    tree += '├── templates/\n';
    tree += '│   ├── deployment.yaml\n';
    tree += '│   ├── service.yaml\n';
    tree += '│   └── ingress.yaml\n';

    subcharts.forEach((subchart, index) => {
      tree += `├── ${subchart.name || `subchart-${index + 1}`}/\n`;
      tree += `│   ├── templates/\n`;
      tree += `│   │   ├── deployment.yaml\n`;
      tree += `│   │   ├── service.yaml\n`;
      tree += `│   │   └── ingress.yaml\n`;
    });

    tree += '└── Chart.yaml\n';
    return tree;
  };

  const TreeView = ({ tree }) => {
    return (
      <pre className="tree-view">
        {tree}
      </pre>
    );
  };

  const steps = [
    {
      title: "Umbrella Chart",
      content: (
        <div className="step-content">
          <label>
            Umbrella Chart Name:
            <input
              type="text"
              placeholder="Enter chart name"
              value={umbrellaChartName}
              onChange={(e) => setUmbrellaChartName(e.target.value)}
              required
            />
          </label>
        </div>
      ),
    },
    {
      title: "Subcharts",
      content: (
        <div className="step-content">
          {subcharts.map((subchart, index) => (
            <div key={index} className="subchart-item">
              <label>
                Name:
                <input
                  type="text"
                  value={subchart.name}
                  onChange={(e) => handleSubchartChange(index, 'name', e.target.value)}
                  required
                />
              </label>
              <label>
                Workload:
                <select
                  value={subchart.workload}
                  onChange={(e) => handleSubchartChange(index, 'workload', e.target.value)}
                >
                  <option value="deployment">Deployment</option>
                  <option value="statefulset">StatefulSet</option>
                  <option value="daemonset">DaemonSet</option>
                </select>
              </label>
            </div>
          ))}
          <button type="button" onClick={handleAddSubchart}>
            Add Subchart
          </button>
        </div>
      ),
    },
    {
      title: "Review & Generate",
      content: (
        <div className="step-content">
          <p>Review your configuration and click "Generate" to proceed.</p>
          <button type="submit">
            Generate Chart
          </button>
        </div>
      ),
    },
  ];

  const settings = {
    dots: true,
    infinite: false,
    speed: 500,
    slidesToShow: 1,
    slidesToScroll: 1,
    beforeChange: (_, next) => setCurrentStep(next),
  };

  return (
    <div>
      {/* Top Navigation Bar */}
      <nav className="top-nav">
        <div className="nav-left">
          <h1>ChartPress</h1>
        </div>
        <div className="nav-right">
          <a href="#home">Home</a>
          <a href="#generate">Generate</a>
          <a href="#documentation">Documentation</a>
          <a href="https://github.com/kriipke/chartpress" target="_blank" rel="noopener noreferrer">GitHub</a>
        </div>
      </nav>

      {/* Define Routes */}
      <Routes>
	<Route path="/documentation" element={<Documentation />} />
      </Routes>

      {/* Main Content */}
      <div className="main-content">
        {/* Wizard Section */}
        <div className="wizard-container">
          <Slider {...settings}>
            {steps.map((step, index) => (
              <div key={index} className="wizard-step">
                <h2>{step.title}</h2>
                {step.content}
              </div>
            ))}
          </Slider>
          {downloadUrl && (
            <div className="download-section">
              <a href={downloadUrl} download={`${umbrellaChartName}.zip`}>
                <button>Download Chart</button>
              </a>
            </div>
          )}
        </div>

        {/* Tree View Section */}
        <TreeView tree={generateTreeStructure()} />
      </div>
    </div>
  );
}

export default App;
