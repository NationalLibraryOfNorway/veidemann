import {Component, Inject, OnInit} from '@angular/core';
import {CrawlJobDetailsComponent} from '..';
import {ReactiveFormsModule, UntypedFormBuilder} from '@angular/forms';
import {AuthService} from '../../../../../core/auth';
import {MAT_DIALOG_DATA, MatDialogModule, MatDialogRef} from '@angular/material/dialog';
import {ConfigDialogData} from '../../../func';
import {ConfigObject} from '../../../../../shared/models/config';
import {MatSlideToggleModule} from '@angular/material/slide-toggle';
import {MetaComponent} from '../../meta/meta.component';
import {MatFormFieldModule} from '@angular/material/form-field';
import {DurationPickerComponent} from '../../durationpicker/duration-picker';
import {FilesizeInputComponent} from '../../filesize-input/filesize-input.component';
import {MatSelectModule} from '@angular/material/select';

@Component({
  selector: 'app-crawljob-dialog',
  templateUrl: './crawljob-dialog.component.html',
  styleUrls: ['./crawljob-dialog.component.css'],
  imports: [
    MatDialogModule,
    MatFormFieldModule,
    MatSelectModule,
    MatSlideToggleModule,
    MetaComponent,
    ReactiveFormsModule,
    DurationPickerComponent,
    FilesizeInputComponent
  ],
  standalone: true
})
export class CrawlJobDialogComponent extends CrawlJobDetailsComponent implements OnInit {

  constructor(protected override fb: UntypedFormBuilder,
              protected override authService: AuthService,
              @Inject(MAT_DIALOG_DATA) public data: ConfigDialogData,
              public dialogRef: MatDialogRef<CrawlJobDialogComponent>) {
    super(fb, authService);
    this.createForm();
    this.configObject = this.data.configObject;
    this.crawlConfigs = this.data.options.crawlConfigs;
    this.crawlScheduleConfigs = this.data.options.crawlScheduleConfigs;
    this.scopeScripts = this.data.options.scopeScripts;
  }

  ngOnInit(): void {
    this.updateForm();
  }

  onDialogClose(): ConfigObject {
    return this.prepareSave();
  }
}
