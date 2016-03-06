'use strict';

require("./node_modules/bootstrap/dist/css/bootstrap.min.css");
require("./main.css");

import React from 'react';
import ReactDOM from 'react-dom';
import _ from 'underscore'
import request from 'browser-request'

const url = "localhost:9999"

const courseInfoQueryTemplate = _.template("http://159.203.112.6:3000/subjects?abbreviation=eq.<%= code %>&select=title,abbreviation,courses{number,title,description}&courses.number=eq.<%= number %>")
export class ChatBox extends React.Component {
  constructor(props) {
    super(props);

    this.state = {
      messages: [],
      name: "Username",
      inputText: "",
      socket: null,
      courses: [],
      top: [],
      courseInfo: null
    };
    this.componentDidMount = this.componentDidMount.bind(this);
    this.handleNewMessage = this.handleNewMessage.bind(this);
    this.handlePostMessage = this.handlePostMessage.bind(this);

    this.handleNameChange = this.handleNameChange.bind(this);
    this.handleInputTextChange = this.handleInputTextChange.bind(this);
    this.starMessageHandler = this.starMessageHandler.bind(this);
    this.messageSendUtil = this.messageSendUtil.bind(this);
    this.handleUpdateTop = this.handleUpdateTop.bind(this);
    this.grabCourseInfo = this.grabCourseInfo.bind(this);
    this.grabOldMessages = this.grabOldMessages.bind(this);


  }
  grabOldMessages(course){
     request("/getSomeMessages/"+course+"/10", (err, res, bod) => {
      if (!err && res.statusCode == 200) {
        let top = JSON.parse(bod)
        let worked = top.map((msg)=>{
          return{
         name: msg.MessageDisplayName,
          id: msg.MessageId,
         text: msg.MessageText,
         stars: msg.NumStars
         }
        })
        // console.log(worked)
        this.setState({messages: worked});
      }
    })
  }


  handleNewMessage(messageInfo) {

    if (messageInfo.name === "__ADMIN__") {
      let starInfo = JSON.parse(messageInfo.text)

      let toChange = _.findWhere(this.state.messages, {id: starInfo.MessageId})
      let index = _.indexOf(this.state.messages, toChange)
      let newMessages = _.clone(this.state.messages)
      newMessages[index].stars = starInfo.NumStars
      this.setState({messages: newMessages})

    } else {

      this.setState({
        messages: [messageInfo].concat(this.state.messages)
      })

    }
  }
  handlePostMessage(e) {
    if (e.keyCode === 13) {
      this.messageSendUtil(this.state.inputText, this.state.name.slice(0, 15))
      this.setState({inputText: ""})
      return false
    }
  }
  handleNameChange(e) {
    this.setState({name: e.target.value});
  }
  handleInputTextChange(e) {
    this.setState({inputText: e.target.value});
  }
  starMessageHandler(key, e) {

    // let changed = _.findWhere(this.state.messages, {id: key})
    this.messageSendUtil("/star " + key.toString(), this.state.name)
    // this.setState({messages:[changed]})
    this.handleUpdateTop()
  }
  messageSendUtil(text, name) {
    var outgoingMessage = {
      MessageText: text,
      MessageDisplayName: name
    };
    this.state.socket.send(JSON.stringify(outgoingMessage));
  }
  handleUpdateTop() {
    request("/getMostStarred/10", (err, res, bod) => {
      if (!err && res.statusCode == 200) {
        let top = JSON.parse(bod)
        this.setState({top: top});
      }
    })
  }
  grabCourseInfo(course) {

    let a = course.split("");
    let isNum = (c) => {
      return (c.charCodeAt(0) < 58)
    }
    let number = _.filter(a, isNum).join("")
    let code = _.reject(a, isNum).join("")

    request(courseInfoQueryTemplate({code, number}), (err, res, bod) => {

      if (!err && res.statusCode == 200) {
        let top = JSON.parse(bod)
        if (top.length===0)
          return
        let courseInfo = {
          subjtitle: top[0].title,
          abbreviation: top[0].abbreviation,
          number: top[0].courses[0].number,
          title: top[0].courses[0].title,
          description: top[0].courses[0].description
        }
        this.setState({courseInfo: courseInfo});
      }
    })

  }
  componentDidMount() {
    let course = window.location.toString().split('?')[1] || "General"
    this.state.socket = new WebSocket("ws://" + window.location.toString().split('/')[2] + "/ws/course/" + course);
    this.grabCourseInfo(course)
    this.grabOldMessages(course)
    this.state.socket.onmessage = (msg) => {
      let parsed = JSON.parse(msg.data)
      let payload = {
        name: parsed.MessageDisplayName,
        id: parsed.MessageId,
        text: parsed.MessageText,
        stars: parsed.NumStars
      }
      this.handleNewMessage(payload)
    }

    this.handleUpdateTop()

    this.state.socket.onclose = function(event) {
      console.log(closed)
    }

    request("/getMyCourses", (err, res, bod) => {
      // console.log("MY COURSES: " + err + res + bod)
      if (!err && res.statusCode == 200) {
        this.setState({courses: JSON.parse(bod)})
      }

    })
  }
  render() {
    return (
      <div id="chatContainer">
        <CourseList options={this.state.courses}/>
        <TopList list={this.state.top} starMessageHandler={this.starMessageHandler} courseInfo={this.state.courseInfo}/>
        <div className='MessageList'>
          <MessageList messages={this.state.messages} starMessageHandler={this.starMessageHandler}/>
        </div>
        <InputForm name ={this.state.name} inputText={this.state.inputText} handleNameChange={this.handleNameChange} handleInputTextChange={this.handleInputTextChange} handlePostMessage={this.handlePostMessage}/>
      </div>
    )
  }
}

export class TopList extends React.Component {
  constructor(props) {
    super(props);
  }
  render() {
    let msgs = this.props.list || []
    var messageNodes = msgs.map((msg) => {
      let payload = {
        name: msg.MessageDisplayName,
        id: msg.MessageId,
        text: msg.MessageText,
        stars: msg.NumStars
      }
      return (<MessageCard key={payload.id} data={payload} starMessageHandler={this.props.starMessageHandler}/>)
    })
    infoCard = ""
    if (this.props.courseInfo != null) {
      var infoCard = (
        <div id="infoCard">
          <h2>
            {this.props.courseInfo.title}</h2>
          <hr/>
          <p>
            {this.props.courseInfo.description}</p>
        </div>
      )
    }
    return (

      <div id="TopList">
        {infoCard}
        <h4>
          Top 10
        </h4>
        <ul id="topmessages">
          {messageNodes}
        </ul>
      </div>
    )
  }
}

export class CourseList extends React.Component {
  constructor(props) {
    super(props);
  }
  render() {
    let CourseNodes = this.props.options.map((course) => {
      return (
        <div key={course} className="CourseButton">
          <a key={course} href={"/room?" + course}>
            <h3>{course}</h3>
          </a>
        </div>
      )
    })
    return (
      <div id="Courses">
        {CourseNodes}
      </div>
    )
  }
}

export class MessageList extends React.Component {
  constructor(props) {
    super(props);
  }
  render() {

    var messageNodes = this.props.messages.map((msg) => {
      return (<MessageCard key={msg.id} data={msg} starMessageHandler={this.props.starMessageHandler}/>)
    })
    return (
      <ul id="messages">
        {messageNodes}
        <a id="bottom" name="bottom"/>
      </ul>
    )
  }
}

export class InputForm extends React.Component {
  constructor(props) {
    super(props);
  }

  render() {
    return (
      <form id="form">
        <input id="desiredName" value={this.props.name} type="text" onChange={this.props.handleNameChange}/>
        <input id="userMessage" value={this.props.inputText} type="text" onChange={this.props.handleInputTextChange} onKeyDown={this.props.handlePostMessage}/>
      </form>
    )
  }
}

export class MessageCard extends React.Component {
  constructor(props) {
    super(props);
  }
  render() {
    let hasStar = this.props.data.stars === undefined
    return (
      <li key ={this.props.data.id} onClick={this.props.starMessageHandler.bind(null, this.props.data.id)}>
        <span className={hasStar
          ? "starPlaceholder"
          : "star"}>
          {this.props.data.stars}
        </span>
        <span className="commentInfo">
          {' '}
          {this.props.data.name}
        </span>
        <span className="commentBody" dangerouslySetInnerHTML={{
          __html: this.props.data.text
        }}/>
      </li>
    )
  }
}
ReactDOM.render(
  <ChatBox url="http://159.203.112.6:3000"/>, document.querySelector("#myApp"));
