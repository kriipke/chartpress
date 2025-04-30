import React, { useState } from 'react'

function App () {
  const [umbrellaChartName, setUmbrellaChartName] = useState('')
  const [subcharts, setSubcharts] = useState([
    { name: '', workload: 'deployment' }
  ])
  const [downloadUrl, setDownloadUrl] = useState('')

  const handleAddSubchart = () => {
    setSubcharts([...subcharts, { name: '', workload: 'deployment' }])
  }

  const handleSubchartChange = (index, field, value) => {
    const updatedSubcharts = [...subcharts]
    updatedSubcharts[index][field] = value
    setSubcharts(updatedSubcharts)
  }

  const handleSubmit = async e => {
    e.preventDefault()
    const config = { umbrellaChartName, subcharts }
    try {
      const response = await fetch('/chartpress/generate', {
        method: 'POST',
        headers: { 'Content-Type': 'application/json' },
        body: JSON.stringify(config)
      })

      if (response.ok) {
        const blob = await response.blob()
        const url = window.URL.createObjectURL(blob)
        setDownloadUrl(url)
        alert(
          'Chart generated successfully! Click "Download Chart" to download the zip file.'
        )
      } else {
        const errorText = await response.text()
        alert(`Error: ${errorText}`)
      }
    } catch (error) {
      console.error('Error submitting configuration:', error)
    }
  }

  return (
    <div style={{ padding: '2rem' }}>
      <h1>ChartPress Wizard</h1>
      <form onSubmit={handleSubmit}>
        <div>
          <label>
            Umbrella Chart Name:
            <input
              type='text'
              value={umbrellaChartName}
              onChange={e => setUmbrellaChartName(e.target.value)}
              required
            />
          </label>
        </div>
        <h2>Subcharts</h2>
        {subcharts.map((subchart, index) => (
          <div key={index} style={{ marginBottom: '1rem' }}>
            <label>
              Name:
              <input
                type='text'
                value={subchart.name}
                onChange={e =>
                  handleSubchartChange(index, 'name', e.target.value)
                }
                required
              />
            </label>
            <label>
              Workload:
              <select
                value={subchart.workload}
                onChange={e =>
                  handleSubchartChange(index, 'workload', e.target.value)
                }
              >
                <option value='deployment'>Deployment</option>
                <option value='statefulset'>StatefulSet</option>
                <option value='daemonset'>DaemonSet</option>
              </select>
            </label>
          </div>
        ))}
        <button type='button' onClick={handleAddSubchart}>
          Add Subchart
        </button>
        <div style={{ marginTop: '1rem' }}>
          <button type='submit'>Generate Chart</button>
        </div>
      </form>
      {downloadUrl && (
        <div style={{ marginTop: '1rem' }}>
          <a href={downloadUrl} download={`${umbrellaChartName}.zip`}>
            <button>Download Chart</button>
          </a>
        </div>
      )}
    </div>
  )
}

export default App
