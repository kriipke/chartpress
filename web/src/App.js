import React, { useState } from 'react';
import chartpressConfig from './chartpress.yaml'; // Import chartpress.yaml for checkbox options

function App() {
  const [currentSlide, setCurrentSlide] = useState(1);
  const [formData, setFormData] = useState({
    umbrellaChartName: '',
    subcharts: [{ name: '', workload: 'deployment' }],
    settings: {}, // To capture selected settings from chartpress.yaml
  });

  const handleFieldChange = (field, value) => {
    setFormData((prev) => ({ ...prev, [field]: value }));
  };

  const handleSubchartChange = (index, field, value) => {
    const updatedSubcharts = [...formData.subcharts];
    updatedSubcharts[index][field] = value;
    setFormData((prev) => ({ ...prev, subcharts: updatedSubcharts }));
  };

  const handleCheckboxChange = (setting, isChecked) => {
    setFormData((prev) => ({
      ...prev,
      settings: { ...prev.settings, [setting]: isChecked },
    }));
  };

  const handleNext = () => {
    setCurrentSlide((prev) => prev + 1);
  };

  const handleBack = () => {
    setCurrentSlide((prev) => prev - 1);
  };

  const handleSubmit = async () => {
    try {
      // Convert formData to the required chartpress.yaml format (if necessary)
      const payload = {
        umbrellaChartName: formData.umbrellaChartName,
        subcharts: formData.subcharts,
        rules: formData.settings, // Assuming settings are directly mapped from checkboxes
      };
  
      console.log('Submitting configuration:', payload);
  
      // Send the POST request to the /chartpress/generate endpoint
      const response = await fetch('/chartpress/generate', {
        method: 'POST',
        headers: { 'Content-Type': 'application/json', 'Accept': 'application/json'},
        body: JSON.stringify(payload),
      })
      .then(response => response.json())
      .then(payload => console.log('Success:', payload))
      .catch(error => console.error('Error:', error));

      let a = document.createElement('a');
      a.href = response["downloadUrl"];
      a.target = '_blank';
      a.click();

    } catch (error) {
      console.error('Error:', error);
      alert('An unexpected error occurred. Please try again.');
    }
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
              value=`${umbrellaChartName}`
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
    <Router>
      <div>
	{/* Top Navigation Bar */}
	<nav className="top-nav">
	  <div className="nav-left">
	    <h1>ChartPress</h1>
	  </div>
	  <div className="nav-right">
	    <a href="/chartpress/">Home</a>
	    <a href="/chartpress/generate">Generate</a>
	    <a href="/chartpress/documentation">Documentation</a>
	    <a href="https://github.com/kriipke/chartpress" target="_blank" rel="noopener noreferrer">GitHub</a>
	  </div>
	</nav>

        {/* Define Routes */}
        <Routes>
          <Route path="/chartpress/documentation" element={<Documentation />} />
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
    </Router>
  );
}

export default App;
