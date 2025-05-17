import React, { useState } from "react";
import SubchartDetails from "./SubchartDetails";
import "./ChartpressHelmChartWizard.css";


export const ChartpressHelmChartWizard = ({ className, ...props }) => {
  // Inside your component:
  const [subcharts, setSubcharts] = useState([]);
  const [loading, setLoading] = useState(false);
  const [result, setResult] = useState(null);

  const handleGenerate = async () => {
    setLoading(true);
    try {
      const response = await fetch("/chartpress/generate", {
        method: "POST",
        headers: { "Content-Type": "application/json" },
        body: JSON.stringify(subcharts), // Send your collected form data
      });
      const data = await response.json();
      setResult(data);
    } catch (err) {
      alert("Error generating chart: " + err.message);
    } finally {
      setLoading(false);
    }
  };
  return (
    <div className={"chartpress-helm-chart-wizard " + className}>
      <div className="container">
        <div className="image">
          <img className="group" src="group0.svg" />
        </div>
        <div className="avatar">
          <img className="rectangle" src="rectangle0.png" />
        </div>
        <div className="chartpress">chartpress </div>
        <div className="header-menu">
          <div className="frame">
            <div className="rectangle2"></div>
            <div className="dashboard">Dashboard </div>
          </div>
          <div className="frame2">
            <div className="reports">Reports </div>
          </div>
          <div className="frame3">
            <div className="projects">Projects </div>
          </div>
        </div>
        <div className="settings-gear">
          <img className="group2" src="group1.svg" />
        </div>
        <div className="question-mark">
          <img className="group3" src="group2.svg" />
        </div>
        <div className="textbox">
          <div className="search">Search ... </div>
          <div className="search2">
            <img className="group4" src="group3.svg" />
          </div>
        </div>
      </div>
      <div className="chart-description">Chart Description </div>
      <div className="textbox-32">
        <div className="helm-chart-name">Helm Chart Name </div>
        <div className="box-3-d-50">
          <img className="group5" src="group4.svg" />
        </div>
      </div>
      <div className="container2">
        <img className="line-6" src="line-60.svg" />
        <div className="header-menu2">
          <div className="frame4">
            <div className="description">Description </div>
          </div>
          <div className="frame5">
            <div className="template-selection">Template Selection </div>
          </div>
          <div className="frame6">
            <div className="subchart-details">Subchart Details </div>
          </div>
          <div className="frame7">
            <div className="structure">Structure </div>
          </div>
        </div>
        <div className="container3">
          <div className="container4">
            <div className="three-way-direction">
              <img className="group6" src="group5.svg" />
            </div>
            <div className="subchart-details2">Subchart Details </div>
          </div>
          <div className="container-3">
            <div className="core-configuration">Core configuration </div>
            <img className="image-20" src="image-200.png" />
            <div className="core-chart">Core Chart </div>
            <div className="defines-primary-chart-dependencies">
              Defines primary chart dependencies{" "}
            </div>
            <div className="button-1">
              <div className="edit-subchart">Edit Subchart </div>
            </div>
          </div>
          <div className="container-4">
            <div className="additional-modules">Additional modules </div>
            <img className="image-20" src="image-201.png" />
            <div className="extension-chart">Extension Chart </div>
            <div className="extends-functionality-and-features">
              Extends functionality and features{" "}
            </div>
            <div className="button-1">
              <div className="edit-subchart">Edit Subchart </div>
            </div>
          </div>
          <div className="container5">
            <div className="d-add">
              <img className="group7" src="group6.svg" />
            </div>
            <div className="add-subchart">Add Subchart </div>
          </div>
          <div className="container6">
            <div className="subchart-name">Subchart Name </div>
            <div className="textbox2">
              <div className="my-subchart">my-subchart </div>
            </div>
            <div className="type">Type </div>
            <div className="textbox3">
              <div className="library">Library </div>
            </div>
            <div className="count">Count </div>
            <div className="container-2">
              <div className="_10">10 </div>
              <div className="arrow-drop-down-large-1">
                <img className="group8" src="group7.svg" />
              </div>
              <div className="arrow-drop-down-large-2">
                <img className="group9" src="group8.svg" />
              </div>
            </div>
          </div>
        </div>
        <div className="container7">
          <div className="container8">
            <div className="three-way-direction">
              <img className="group10" src="group9.svg" />
            </div>
            <div className="template-selection2">Template Selection </div>
          </div>
          <div className="container9">
            <div className="language">Language </div>
            <div className="textbox4">
              <div className="all">All </div>
            </div>
            <div className="functionality">Functionality </div>
            <div className="textbox5">
              <div className="all2">All </div>
            </div>
            <div className="button">
              <div className="reset">Reset </div>
            </div>
            <div className="button2">
              <div className="apply">Apply </div>
            </div>
          </div>
          <div className="container-5">
            <img className="image-135" src="image-1350.png" />
            <div className="chart-wizard">Chart Wizard </div>
            <div className="define-chart-metadata">Define chart metadata </div>
            <div className="start-by-setting-basic-properties">
              Start by setting basic properties{" "}
            </div>
            <div className="button-43">
              <div className="start-setup">Start Setup </div>
            </div>
          </div>
          <div className="container-6">
            <img className="image-135" src="image-1351.png" />
            <div className="subchart-setup">Subchart Setup </div>
            <div className="specify-subcharts">Specify subcharts </div>
            <div className="add-dependencies-seamlessly">
              Add dependencies seamlessly{" "}
            </div>
            <div className="button-432">
              <div className="add-subcharts">Add Subcharts </div>
            </div>
          </div>
          <div className="container-7">
            <img className="image-135" src="image-1352.png" />
            <div className="template-picker">Template Picker </div>
            <div className="choose-a-template">Choose a template </div>
            <div className="pick-from-pre-made-options">
              Pick from pre-made options{" "}
            </div>
            <div className="button-433">
              <div className="select-template">Select Template </div>
            </div>
          </div>
          <div className="container-8">
            <img className="image-135" src="image-1353.png" />
            <div className="values-setup">Values Setup </div>
            <div className="define-chart-values">Define chart values </div>
            <div className="configure-form-fields">Configure form fields </div>
            <div className="button-434">
              <div className="configure-values">Configure Values </div>
            </div>
          </div>
          <div className="container-9">
            <img className="image-135" src="image-1354.png" />
            <div className="setup-preview">Setup Preview </div>
            <div className="preview-configurations">
              Preview configurations{" "}
            </div>
            <div className="verify-setup-details">Verify setup details </div>
            <div className="button-435">
              <div className="preview-chart">Preview Chart </div>
            </div>
          </div>
          <div className="container-10">
            <img className="image-135" src="image-1355.png" />
            <div className="deployment">Deployment </div>
            <div className="finalize-and-deploy">Finalize and deploy </div>
            <div className="complete-chart-creation">
              Complete chart creation{" "}
            </div>
            <div className="button-436">
              <div className="deploy-chart">Deploy Chart </div>
            </div>
          </div>
        </div>
        <div className="container-11">
          <div className="container-12">
            <img className="image-3" src="image-30.png" />
          </div>
          <div className="favorite-1">
            <img className="group11" src="group10.svg" />
          </div>
          <div className="cache">Cache </div>
          <div className="deployment2">Deployment </div>
          <div className="oci-your-repo-cache">oci://your.repo/cache </div>
          <div className="pin-3-2">
            <img className="group12" src="group11.svg" />
          </div>
          <div className="networked">Networked </div>
          <div className="bookmark-1">
            <img className="group13" src="group12.svg" />
          </div>
        </div>
        <div className="subchart-management">
          <div className="subchart-information-container">
            <div className="image-container">
              <img className="subchart-image" src="subchart-image0.png" />
            </div>
            <div className="favorite-icon">
              <img className="group14" src="group13.svg" />
            </div>
            <div className="web-label">Web </div>
            <div className="deployment-label">Deployment </div>
            <div className="web-repository-link">oci://your.repo/web </div>
            <div className="forked-repository-icon">
              <img className="group15" src="group14.svg" />
            </div>
            <div className="networked-label">Networked </div>
            <div className="bookmark-icon">
              <img className="group16" src="group15.svg" />
            </div>
          </div>
          <div className="table-2">
            <div className="header">
              <div className="frame8">
                <div className="type2">Type </div>
              </div>
              <div className="frame9">
                <div className="image2">Image </div>
              </div>
              <div className="frame10"></div>
              <div className="frame11">
                <div className="tag">Tag </div>
              </div>
              <div className="frame12">
                <div className="description2">Description </div>
              </div>
              <div className="frame13">
                <div className="subchart">Subchart </div>
              </div>
            </div>
            <div className="row">
              <div className="frame14">
                <div className="checkbox">
                  <div className="frame15">
                    <div className="rectangle3"></div>
                  </div>
                </div>
              </div>
              <div className="frame16">
                <div className="cache2">Cache </div>
              </div>
              <div className="frame17">
                <div className="tag2">
                  <div className="frame18">
                    <div className="new-tag">New tag </div>
                  </div>
                </div>
              </div>
              <div className="frame19">
                <img className="image3" src="image2.png" />
              </div>
              <div className="frame20">
                <div className="pen">
                  <img className="group17" src="group16.svg" />
                </div>
                <div className="kriipke-cache">kriipke/cache </div>
              </div>
              <div className="frame21">
                <div className="minim-ullamco-duis-minim-consequat-officia-mollit-sit-officia">
                  Minim ullamco duis minim consequat officia mollit sit officia{" "}
                </div>
              </div>
            </div>
            <div className="row2">
              <div className="frame19">
                <img className="image3" src="image3.png" />
              </div>
              <div className="frame16">
                <div className="api">API </div>
              </div>
              <div className="frame14">
                <div className="checkbox">
                  <div className="frame15">
                    <div className="rectangle3"></div>
                  </div>
                </div>
              </div>
              <div className="frame20">
                <div className="kriipke-api">kriipke/api </div>
                <div className="pen">
                  <img className="group18" src="group17.svg" />
                </div>
              </div>
              <div className="frame17">
                <div className="tag2">
                  <div className="frame18">
                    <div className="new-tag">New tag </div>
                  </div>
                </div>
              </div>
              <div className="frame21">
                <div className="occaecat-amet-deserunt-magna-elit-ex-esse-aliquip-enim-f">
                  Occaecat amet deserunt magna elit ex esse aliquip. Enim f{" "}
                </div>
              </div>
            </div>
            <div className="row3">
              <div className="frame14">
                <div className="checkbox">
                  <div className="frame15">
                    <div className="rectangle3"></div>
                  </div>
                </div>
              </div>
              <div className="frame17">
                <div className="tag2">
                  <div className="frame18">
                    <div className="new-tag">New tag </div>
                  </div>
                </div>
              </div>
              <div className="frame16">
                <div className="web">Web </div>
              </div>
              <div className="frame19">
                <img className="image3" src="image4.png" />
              </div>
              <div className="frame20">
                <div className="pen">
                  <img className="group19" src="group18.svg" />
                </div>
                <div className="kriipke-web">kriipke/web </div>
              </div>
              <div className="frame21">
                <div className="ad-consectetur-tempor-laboris-magna-in-adipisicing-aute-si">
                  Ad consectetur tempor laboris magna in adipisicing aute si{" "}
                </div>
              </div>
            </div>
          </div>
          <div className="card-api"></div>
          <div className="icon-bg"></div>
          <div className="favorite-icon-2">
            <img className="group20" src="group19.svg" />
          </div>
          <img className="subchart-image-2" src="subchart-image-20.png" />
          <div className="api-label">API </div>
          <div className="bookmark-icon-2">
            <img className="group21" src="group20.svg" />
          </div>
          <div className="networked-label-2">Networked </div>
          <div className="api-repository-link">oci://your.repo/api </div>
          <div className="forked-repository-icon-2">
            <img className="group22" src="group21.svg" />
          </div>
          <div className="stateful-set-label">StatefulSet </div>
          <div className="settings">Settings </div>
          <div className="components">Components </div>
          <div className="subchart-selection-checkbox">
            <div className="frame22">
              <div className="rectangle4"></div>
              <img className="frame23" src="frame38.svg" />
              <div className="library2">Library </div>
            </div>
            <div className="frame24">
              <div className="rectangle5"></div>
              <div className="chart">Chart </div>
            </div>
          </div>
          <div className="subchart-selection-checkbox2">
            <div className="frame24">
              <div className="rectangle5"></div>
              <div className="chart">Chart </div>
            </div>
            <div className="frame22">
              <div className="rectangle4"></div>
              <img className="frame25" src="frame42.svg" />
              <div className="library2">Library </div>
            </div>
          </div>
          <div className="subchart-selection-checkbox3">
            <div className="frame22">
              <div className="rectangle4"></div>
              <img className="frame26" src="frame44.svg" />
              <div className="library2">Library </div>
            </div>
            <div className="frame24">
              <div className="rectangle5"></div>
              <div className="chart">Chart </div>
            </div>
          </div>
          <div className="subchart-selection-checkbox4">
            <div className="frame22">
              <div className="rectangle4"></div>
              <img className="frame27" src="frame47.svg" />
              <div className="library2">Library </div>
            </div>
            <div className="frame24">
              <div className="rectangle5"></div>
              <div className="chart">Chart </div>
            </div>
          </div>
          <div className="button-27" onClick={handleGenerate} style={{ cursor: "pointer" }}>
            <div className="generate">{loading ? "Generating..." : "GENERATE"}</div>
          </div>
        </div>
        <div className="textarea">
          <div className="input-text">Input text </div>
          <div className="resizing-handle">
            <img className="group23" src="group22.svg" />
          </div>
        </div>
        <div className="d-add2">
          <img className="group24" src="group23.svg" />
        </div>
        <div className="add-subchart2">Add Subchart </div>
        <div className="subchart-selection-checkbox5">
          <div className="frame22">
            <div className="rectangle4"></div>
            <img className="frame28" src="frame50.svg" />
            <div className="library2">Library </div>
          </div>
          <div className="frame24">
            <div className="rectangle5"></div>
            <div className="chart">Chart </div>
          </div>
        </div>
        <div className="subchart-selection-checkbox6">
          <div className="frame24">
            <div className="rectangle5"></div>
            <div className="chart">Chart </div>
          </div>
          <div className="frame22">
            <div className="rectangle4"></div>
            <img className="frame29" src="frame54.svg" />
            <div className="library2">Library </div>
          </div>
        </div>
        <div className="configuration">Configuration </div>
      </div>
    </div>
    <div>
      <SubchartDetails subcharts={subcharts} setSubcharts={setSubcharts} />
    </div>
  );
};
