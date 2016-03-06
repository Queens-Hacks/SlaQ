'use strict';

require("./node_modules/bootstrap/dist/css/bootstrap.min.css");
require("./main.css");

import React from 'react';
import ReactDOM from 'react-dom';
import _ from 'underscore'
import request from 'browser-request'

const url = "localhost:9999"

export class ChatBox extends React.Component {
  constructor(props) {
    super(props);

    this.state = {
      messages: [],
      name: "Username",
      inputText: "",
      socket: null,
      courses: []
    };
    this.componentDidMount = this.componentDidMount.bind(this);
    this.handleNewMessage = this.handleNewMessage.bind(this);
    this.handlePostMessage = this.handlePostMessage.bind(this);

    this.handleNameChange = this.handleNameChange.bind(this);
    this.handleInputTextChange = this.handleInputTextChange.bind(this);
    this.starMessageHandler = this.starMessageHandler.bind(this);
    this.messageSendUtil = this.messageSendUtil.bind(this);

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
    console.log("starring " + key.toString())
    this.messageSendUtil("/star " + key.toString(), this.state.name)
    // this.setState({messages:[changed]})
  }

  messageSendUtil(text, name) {
    var outgoingMessage = {
      MessageText: text,
      MessageDisplayName: name
    };
    this.state.socket.send(JSON.stringify(outgoingMessage));
  }
  componentDidMount() {
    let course = window.location.toString().split('?')[1]
    this.state.socket = new WebSocket("ws://" + window.location.toString().split('/')[2] + "/ws/course/" + course);

    this.state.socket.onmessage = (msg) => {
      let parsed = JSON.parse(msg.data)
      let payload = {
        name: parsed.MessageDisplayName,
        id: parsed.MessageId,
        text: parsed.MessageText,
        stars: 0
      }
      this.handleNewMessage(payload)
    }

    this.state.socket.onclose = function(event) {
      console.log(closed)
    }
    // request({
    //   method: 'POST',
    //   url: 'http://localhost:9999/login'
    // }, on_response)

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
        <div className='MessageList'>
          <MessageList messages={this.state.messages} starMessageHandler={this.starMessageHandler}/>
        </div>
        <InputForm name ={this.state.name} inputText={this.state.inputText} handleNameChange={this.handleNameChange} handleInputTextChange={this.handleInputTextChange} handlePostMessage={this.handlePostMessage}/>
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

    var messageNodes = this.props.messages.map((course) => {
      return (<MessageCard key={course.id} data={course} starMessageHandler={this.props.starMessageHandler}/>)
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
    let hasStar = this.props.data.stars === 0
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
