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
        headers: {
          'Content-Type': 'application/json',
          'Accept': 'application/json',
        },
        body: JSON.stringify(payload)
      })
      .then(response => response.json())
      .then(payload => console.log('Success:', payload))
      .catch(error => console.error('Error:', error));

      /INITIATE DOWNLOAD
      let a = document.createElement('a');
      a.href = response["downloadUrl"];
      a.target = '_blank';
      a.click();

    } catch (error) {
      console.error('Error:', error);
      alert('An unexpected error occurred. Please try again.');
    }
  };

  return (
    <div style={{ padding: '2rem' }}>
      <h1>ChartPress Wizard</h1>
      {currentSlide === 1 && (
        <div>
          <h2>Umbrella Chart Configuration</h2>
          <label>
            Umbrella Chart Name:
            <input
              type="text"
              value={formData.umbrellaChartName}
              onChange={(e) => handleFieldChange('umbrellaChartName', e.target.value)}
              required
            />
          </label>
          <h3>Subcharts</h3>
          {formData.subcharts.map((subchart, index) => (
            <div key={index} style={{ marginBottom: '1rem' }}>
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
          <button type="button" onClick={() => handleFieldChange('subcharts', [...formData.subcharts, { name: '', workload: 'deployment' }])}>
            Add Subchart
          </button>
          <div style={{ marginTop: '1rem' }}>
            <button onClick={handleNext}>Next</button>
          </div>
        </div>
      )}

      {currentSlide === 2 && (
        <div>
          <h2>ChartPress Settings</h2>
          {Object.keys(chartpressConfig.rules || {}).map((setting) => (
            <div key={setting}>
              <label>
                <input
                  type="checkbox"
                  checked={!!formData.settings[setting]}
                  onChange={(e) => handleCheckboxChange(setting, e.target.checked)}
                />
                {setting}
              </label>
            </div>
          ))}
          <div style={{ marginTop: '1rem' }}>
            <button onClick={handleBack}>Back</button>
            <button onClick={handleSubmit}>Submit</button>
          </div>
        </div>
      )}
    </div>
  );
}

export default App;
