import {Component, Input} from '@angular/core';
import {ConfigObject, RobotsPolicy} from '../../../../../shared/models/config';
import {DurationFormatPipe} from '../../../../../shared/pipes/duration-format.pipe';
import {MatCheckbox} from '@angular/material/checkbox';
import {MatLabel} from '@angular/material/form-field';
import {FormsModule} from '@angular/forms';
import {LayoutDirective} from '@ngbracket/ngx-layout';

@Component({
  selector: 'app-politenessconfig-preview',
  templateUrl: './politenessconfig-preview.component.html',
  styleUrls: ['./politenessconfig-preview.component.css'],
  imports: [
    DurationFormatPipe,
    FormsModule,
    LayoutDirective,
    MatCheckbox,
    MatLabel
  ],
  standalone: true
})
export class PolitenessconfigPreviewComponent {
  readonly RobotsPolicy = RobotsPolicy;

  @Input()
  configObject: ConfigObject;

  constructor() {
  }


}
